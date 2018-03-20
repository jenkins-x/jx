package jenkins

import (
	"fmt"
	"io"
	"net/url"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
)

// ImportProject imports a MultiBranchProject into Jeknins for the given git URL
func ImportProject(out io.Writer, jenk *gojenkins.Jenkins, gitURL string, dir string, jenkinsfile string, branchPattern, credentials string, failIfExists bool, gitProvider gits.GitProvider, authConfigSvc auth.AuthConfigService) error {
	if gitURL == "" {
		return fmt.Errorf("No Git repository URL found!")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", gitURL, err)
	}

	if branchPattern == "" {
		fork, err := gits.GitIsFork(gitProvider, gitInfo, dir)
		if err != nil {
			return fmt.Errorf("No branch pattern specified and could not determine if the git repository is a fork: %s", err)
		}
		if fork {
			// lets figure out which branches to enable for a fork
			branch, err := gits.GitGetBranch(dir)
			if err != nil {
				return fmt.Errorf("Failed to get current branch in dir %s: %s", dir, err)
			}
			if branch == "" {
				return fmt.Errorf("Failed to get current branch in dir %s", dir)
			}
			// TODO do we need to scape any wacky characters to make it a valid branch pattern?
			branchPattern = branch
			fmt.Fprintf(out, "No branch pattern specified and this repository appears to be a fork so defaulting the branch patterns to run CI / CD on to: %s\n", branchPattern)
		} else {
			branchPattern = DefaultBranchPattern
		}
	}

	if credentials == "" {
		credentials = DefaultJenkinsCredentialsPrefix + "git"
	}
	_, err = jenk.GetCredential(credentials)
	if err != nil {
		config := authConfigSvc.Config()
		server := config.GetOrCreateServer(gitInfo.Host)
		user, err := config.PickServerUserAuth(server, "user name for the Jenkins Pipeline", false)
		if err != nil {
			return err
		}
		err = jenk.CreateCredential(credentials, user.Username, user.ApiToken)

		if err != nil {
			return fmt.Errorf("error creating jenkins credential %s at %s %v", credentials, jenk.BaseURL(), err)
		}
	}

	org := gitInfo.Organisation
	folder, err := jenk.GetJob(org)
	if err != nil {
		// could not find folder so lets try create it
		jobUrl := util.UrlJoin(jenk.BaseURL(), jenk.GetJobURLPath(org))
		folderXml := CreateFolderXml(jobUrl, org)
		//fmt.Fprintf(out, "XML: %s\n", folderXml)
		err = jenk.CreateJobWithXML(folderXml, org)
		if err != nil {
			return fmt.Errorf("Failed to create the %s folder in jenkins: %s", org, err)
		}
		//fmt.Fprintf(out, "Created Jenkins folder: %s\n", org)
	} else {
		c := folder.Class
		if c != "com.cloudbees.hudson.plugins.folder.Folder" {
			fmt.Fprintf(out, "Warning the folder %s is of class %s", org, c)
		}
	}
	projectXml := CreateMultiBranchProjectXml(gitInfo, gitProvider, credentials, branchPattern, jenkinsfile)
	jobName := gitInfo.Name
	job, err := jenk.GetJobByPath(org, jobName)
	if err == nil {
		if failIfExists {
			return fmt.Errorf("Job already exists in Jenkins at %s", job.Url)
		} else {
			fmt.Fprintf(out, "Job already exists in Jenkins at %s\n", job.Url)
		}
	} else {
		//fmt.Fprintf(out, "Creating MultiBranchProject %s from XML: %s\n", jobName, projectXml)
		err = jenk.CreateFolderJobWithXML(projectXml, org, jobName)
		if err != nil {
			return fmt.Errorf("Failed to create MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
		}
		job, err = jenk.GetJobByPath(org, jobName)
		if err != nil {
			return fmt.Errorf("Failed to find the MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
		}
		fmt.Fprintf(out, "Created Jenkins Project: %s\n", util.ColorInfo(job.Url))
		fmt.Fprintln(out)
		fmt.Fprintf(out, "You can view the pipelines via: %s\n", util.ColorInfo("jx get pipelines"))
		fmt.Fprintf(out, "Open the Jenkins console via    %s\n", util.ColorInfo("jx console"))
		fmt.Fprintf(out, "Browse the pipeline log via:    %s\n", util.ColorInfo(fmt.Sprintf("jx get build logs %s", gitInfo.PipelinePath())))
		fmt.Fprintf(out, "View pipeline activity via:     %s\n", util.ColorInfo("jx get activity"))
		fmt.Fprintf(out, "When the pipeline is complete:  %s\n", util.ColorInfo("jx get applications"))
		fmt.Fprintln(out)
		fmt.Fprintf(out, "For more help on available commands see: %s\n", util.ColorInfo("http://jenkins-x.io/developing/browsing/"))
		fmt.Fprintln(out)
		fmt.Fprintf(out, util.ColorStatus("Note that your first pipeline may take a few minutes to start while the necessary docker images get downloaded!\n\n"))

		params := url.Values{}
		err = jenk.Build(job, params)
		if err != nil {
			return fmt.Errorf("Failed to trigger job %s due to %s", job.Url, err)
		}

	}

	// lets disable webhooks for gitea
	if gitProvider.IsGitea() {
		return nil
	}
	// register the webhook
	suffix := gitProvider.JenkinsWebHookPath(gitURL, "")
	webhookUrl := util.UrlJoin(jenk.BaseURL(), suffix)
	webhook := &gits.GitWebHookArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		URL:   webhookUrl,
	}
	return gitProvider.CreateWebHook(webhook)
}
