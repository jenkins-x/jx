package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StepCustomPipelineOptions contains the command line arguments for this command
type StepCustomPipelineOptions struct {
	StepOptions

	MultiBranchProject bool
	Dir                string
	Jenkinsfile        string
	JenkinsPath        string
	JenkinsSelector    JenkinsSelectorOptions
}

var (
	stepCustomPipelineLong = templates.LongDesc(`
		This pipeline step lazily creates a Pipeline job inside a custom Jenkins App and then triggers it

`)

	stepCustomPipelineExample = templates.Examples(`
		# triggers the Jenkinsfile in the current directory in the custom Jenkins App
		jx step custom pipeline
`)
)

func NewCmdStepCustomPipeline(commonOpts *CommonOptions) *cobra.Command {
	options := StepCustomPipelineOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
		JenkinsSelector: JenkinsSelectorOptions{
			UseCustomJenkins: true,
		},
	}
	cmd := &cobra.Command{
		Use:     "custom pipeline",
		Short:   "Triggers a pipeline in a custom Jenkins server using the local Jenkinsfile",
		Long:    stepCustomPipelineLong,
		Example: stepCustomPipelineExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.JenkinsSelector.CustomJenkinsName, "jenkins-name", "j", "", "The name of the custom Jenkins App if you don't wish to use the default execution engine in Jenkins X")

	cmd.Flags().BoolVarP(&options.MultiBranchProject, "multi-branch-project", "", false, "Use a Multi Branch Project in Jenkins")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the Jenkisnfile inside")
	cmd.Flags().StringVarP(&options.Jenkinsfile, "jenkinsfile", "f", jenkinsfile.Name, "The name of the Jenkinsfile to use")
	cmd.Flags().StringVarP(&options.JenkinsPath, "jenkins-path", "p", "", "The Jenkins folder path to create the pipeline inside. If not specified it defaults to the git 'owner/repoName/branch'")
	return cmd
}

func (o *StepCustomPipelineOptions) Run() error {
	jenkinsClient, err := o.CreateCustomJenkinsClient(&o.JenkinsSelector)
	if err != nil {
		return err
	}
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if o.Jenkinsfile == "" {
		o.Jenkinsfile = jenkinsfile.Name
	}
	jenkinsfileName := filepath.Join(o.Dir, o.Jenkinsfile)
	exists, err := util.FileExists(jenkinsfileName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s does not exist", jenkinsfileName)
	}

	gitInfo, err := o.FindGitInfo(o.Dir)
	if err != nil {
		return err
	}

	branch, err := o.Git().Branch(o.Dir)
	if err != nil {
		return err
	}
	if branch == "" {
		branch = "master"
	}

	if o.JenkinsPath == "" {
		o.JenkinsPath = fmt.Sprintf("%s/%s/%s", gitInfo.Organisation, gitInfo.Name, branch)
	}

	paths := strings.Split(o.JenkinsPath, "/")
	last := len(paths) - 1
	for i, path := range paths {
		folderPath := paths[0 : i+1]
		folder, err := jenkinsClient.GetJobByPath(folderPath...)
		fullPath := util.UrlJoin(folderPath...)
		jobURL := util.UrlJoin(jenkinsClient.BaseURL(), fullPath)

		if i < last {
			// lets ensure there's a folder
			err = o.retry(3, time.Second*10, func() error {
				if err != nil {
					folderXml := jenkins.CreateFolderXml(jobURL, path)
					if i == 0 {
						err = jenkinsClient.CreateJobWithXML(folderXml, path)
						if err != nil {
							return errors.Wrapf(err, "failed to create the %s folder at %s in Jenkins", path, jobURL)
						}
					} else {
						folders := strings.Join(paths[0:i], "/job/")
						err = jenkinsClient.CreateFolderJobWithXML(folderXml, folders, path)
						if err != nil {
							return errors.Wrapf(err, "failed to create the %s folder in folders %s at %s in Jenkins", path, folders, jobURL)
						}
					}
				} else {
					c := folder.Class
					if c != "com.cloudbees.hudson.plugins.folder.Folder" {
						log.Warnf("Warning the folder %s is of class %s", jobURL, c)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			gitUrl := gitInfo.HttpCloneURL()
			log.Infof("Using git URL %s and branch %s\n", util.ColorInfo(gitUrl), util.ColorInfo(branch))

			err = o.retry(3, time.Second*10, func() error {
				if err != nil {
					pipelineXml := jenkins.CreatePipelineXml(gitUrl, branch, o.Jenkinsfile)
					if i == 0 {
						err = jenkinsClient.CreateJobWithXML(pipelineXml, path)
						if err != nil {
							return errors.Wrapf(err, "failed to create the %s pipeline at %s in Jenkins", path, jobURL)
						}
					} else {
						folders := strings.Join(paths[0:i], "/job/")
						err = jenkinsClient.CreateFolderJobWithXML(pipelineXml, folders, path)
						if err != nil {
							return errors.Wrapf(err, "failed to create the %s pipeline in folders %s at %s in Jenkins", path, folders, jobURL)
						}
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			job, err := jenkinsClient.GetJobByPath(paths...)
			if err != nil {
				return err
			}
			job.Url = switchJenkinsBaseURL(job.Url, jenkinsClient.BaseURL())
			jobPath := strings.Join(paths, "/")
			log.Infof("triggering pipeline job %s\n", util.ColorInfo(jobPath))
			err = jenkinsClient.Build(job, url.Values{})
			if err != nil {
				return errors.Wrapf(err, "failed to trigger job %s", jobPath)
			}
		}
	}
	return nil
}
