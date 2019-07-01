package opts

import (
	"fmt"
	"net/url"
	"time"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// ImportProjectIntoJenkins imports a MultiBranchProject into Jenkins for the given git URL
func (o *CommonOptions) ImportProjectIntoJenkins(gitURL string, dir string, jenkinsfile string, branchPattern string, failIfExists bool,
	gitProvider gits.GitProvider, isEnvironment bool, batchMode bool) error {
	if gitURL == "" {
		return fmt.Errorf("no Git repository URL found!")
	}

	jc, err := o.JenkinsClient()
	if err != nil {
		return errors.Wrap(err, "creating Jenkins client")
	}

	if branchPattern == "" {
		branchPattern, err = o.getBranchPattern(gitProvider, dir)
		if err != nil {
			return errors.Wrapf(err, "getting the branch pattern from %q directory", dir)
		}
	}

	if err := o.updateJenkinsCredentials(jc, gitProvider); err != nil {
		return errors.Wrapf(err, "updating Jenkins credentials from git provider %q", gitProvider.Kind())
	}

	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return fmt.Errorf("parsing the Git URL %s due to: %s", gitURL, err)
	}
	org := gitInfo.Organisation
	err = o.Retry(10, time.Second*10, func() error {
		folder, err := jc.GetJob(org)
		if err != nil {
			// could not find folder so lets try create it
			jobUrl := util.UrlJoin(jc.BaseURL(), jc.GetJobURLPath(org))
			folderXML := jenkins.CreateFolderXML(jobUrl, org)
			if err := jc.CreateJobWithXML(folderXML, org); err != nil {
				return errors.Wrapf(err, "creating folder %q in Jenkins", org)
			}
		} else {
			c := folder.Class
			if c != "com.cloudbees.hudson.plugins.folder.Folder" {
				log.Logger().Warnf("Warning the folder %s is of class %s", org, c)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "creating Jenkins folder")
	}

	server := gitProvider.Server()
	err = o.Retry(10, time.Second*10, func() error {
		projectXml := jenkins.CreateMultiBranchProjectXml(gitInfo, gitProvider, server.Name, branchPattern, jenkinsfile)
		jobName := gitInfo.Name
		job, err := jc.GetJobByPath(org, jobName)
		if err == nil {
			if failIfExists {
				return fmt.Errorf("job already exists in Jenkins at %s", job.Url)
			} else {
				log.Logger().Infof("job already exists in Jenkins at %s", job.Url)
			}
		} else {
			err = jc.CreateFolderJobWithXML(projectXml, org, jobName)
			if err != nil {
				return errors.Wrapf(err, "creating the multi-branch project job %q in folder %q", jobName, org)
			}
			job, err = jc.GetJobByPath(org, jobName)
			if err != nil {
				return errors.Wrapf(err, "finding the multi-branch project job %q in folder %q", jobName, org)
			}
			log.Logger().Infof("Created Jenkins Project: %s", util.ColorInfo(job.Url))
			o.LogImportedProject(isEnvironment, gitInfo)

			params := url.Values{}
			if err := jc.Build(job, params); err != nil {
				return errors.Wrapf(err, "triggering job %q", job.Url)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "creating multi-branch project")
	}

	suffix := gitProvider.JenkinsWebHookPath(gitURL, "")
	jenkinsBaseURL := o.ExternalJenkinsBaseURL
	if jenkinsBaseURL == "" {
		jenkinsBaseURL = jc.BaseURL()
	}
	webhookUrl := util.UrlJoin(jenkinsBaseURL, suffix)
	webhook := &gits.GitWebHookArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo,
		URL:   webhookUrl,
	}
	return gitProvider.CreateWebHook(webhook)
}

// GenerateProwConfig regenerates the Prow configurations after we have created a SourceRepository
func (o *CommonOptions) GenerateProwConfig(currentNamespace string, devEnv *v1.Environment, sr *v1.SourceRepository) error {
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	jxClient, _, err := o.JXClient()
	if err != nil {
		return err
	}

	defaultSchedulerName := devEnv.Spec.TeamSettings.DefaultScheduler.Name
	config, plugins, err := pipelinescheduler.GenerateProw(false, true, jxClient, currentNamespace, defaultSchedulerName, devEnv, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to update the Prow 'config' and 'plugins' ConfigMaps after adding the new SourceRepository %s", sr.Name)
	}
	err = pipelinescheduler.ApplyDirectly(kubeClient, currentNamespace, config, plugins)
	if err != nil {
		return errors.Wrapf(err, "applying Prow config in namespace %s", currentNamespace)
	}
	log.Logger().Infof("regenerated Prow configuration with the extra SourceRepository: %s\n", util.ColorInfo(sr.Name))
	return nil
}

// getBranchPattern determines the branch pattern from given directory
func (o *CommonOptions) getBranchPattern(gitProvider gits.GitProvider, dir string) (string, error) {
	patterns, err := o.TeamBranchPatterns()
	if err != nil {
		return "", errors.Wrap(err, "getting the team's branch patterns")
	}
	branchPattern := patterns.DefaultBranchPattern
	if branchPattern != "" {
		return branchPattern, nil
	}

	server := gitProvider.Server()
	log.Logger().Infof("Querying if the repository is a fork at %s with kind %s", server.URL, gitProvider.Kind())
	fork, err := o.Git().IsFork(dir)
	if err != nil {
		return "", errors.Wrapf(err, "could not determine if the Git repository from dir %q is a fork", dir)
	}
	if fork {
		// lets figure out which branches to enable for a fork
		branch, err := o.Git().Branch(dir)
		if err != nil {
			return "", errors.Wrapf(err, "getting the current branch from dir %q", dir)
		}
		if branch == "" {
			return "", fmt.Errorf("cannot get the current branch from dir %s", dir)
		}
		// TODO do we need to scape any wacky characters to make it a valid branch pattern?
		branchPattern = branch
		log.Logger().Infof("No branch pattern specified and this repository appears to be a fork so defaulting the branch patterns to run CI/CD on to: %s", branchPattern)
	} else {
		branchPattern = jenkins.BranchPattern(gitProvider.Kind())
	}
	return branchPattern, nil
}

// updateJenkinsCredentials updates the credentials in Jenkins from current git provider server configuration
func (o *CommonOptions) updateJenkinsCredentials(jenkinsClient gojenkins.JenkinsClient, gitProvider gits.GitProvider) error {
	server := gitProvider.Server()
	user, err := server.GetCurrentUser()
	if err != nil {
		return errors.Wrapf(err, "getting the current user for git provider %q", gitProvider.Kind())
	}

	_, err = jenkinsClient.GetCredential(server.Name)
	if err == nil {
		// TODO update the credentials when they exist
		return nil
	}

	err = jenkinsClient.CreateCredential(server.Name, user.Username, user.ApiToken)
	if err != nil {
		return errors.Wrapf(err, "creating Jenkins credentials from current git provider %q credentials", server.Name)
	}

	log.Logger().Infof("Created Jenkins credential %q for host %q user %q",
		util.ColorInfo(server.Name), util.ColorInfo(server.URL), util.ColorInfo(user.Username))

	return nil
}

// LogImportedProject logs more details for an imported project
func (o *CommonOptions) LogImportedProject(isEnvironment bool, gitInfo *gits.GitRepository) {
	log.Blank()
	if !isEnvironment {
		log.Logger().Infof("Watch pipeline activity via:    %s", util.ColorInfo(fmt.Sprintf("jx get activity -f %s -w", gitInfo.Name)))
		log.Logger().Infof("Browse the pipeline log via:    %s", util.ColorInfo(fmt.Sprintf("jx get build logs %s", gitInfo.PipelinePath())))
		log.Logger().Infof("You can list the pipelines via: %s", util.ColorInfo("jx get pipelines"))
		log.Logger().Infof("When the pipeline is complete:  %s", util.ColorInfo("jx get applications"))
		log.Blank()
		log.Logger().Infof("For more help on available commands see: %s", util.ColorInfo("https://jenkins-x.io/developing/browsing/"))
		log.Blank()
	}
	log.Logger().Info(util.ColorStatus("Note that your first pipeline may take a few minutes to start while the necessary images get downloaded!\n"))
}
