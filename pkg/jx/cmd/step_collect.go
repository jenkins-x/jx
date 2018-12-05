package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepCollect contains the command line flags
type StepCollectOptions struct {
	StepOptions
	Provider string
	GitHubPagesStepCollectOptions
	HttpStepCollectOptions
	Pattern    []string
	Classifier string
}

type GitHubPagesStepCollectOptions struct {
}

type HttpStepCollectOptions struct {
	Destination string
}

type CollectProviderKind string

const gitHubRepoPattern = `^https?:\/\/github.com\/([a-zA-Z1-9-]*)\/([a-zA-Z1-9-]*)\.git$`

const (
	envVarBranchName = "BRANCH_NAME"
	envVarSourceUrl  = "SOURCE_URL"
)

const ghPagesBranchName = "gh-pages"

const (
	GitHubPagesCollectProviderKind CollectProviderKind = "GitHub"
	HttpCollectProviderKind        CollectProviderKind = "Http"
)

var CollectProvidersKinds = []string{
	string(GitHubPagesCollectProviderKind),
	string(HttpCollectProviderKind),
}

var (
	StepCollectLong = templates.LongDesc(`
		This pipeline step collects the specified files that need storing from the build 
`)

	StepCollectExample = templates.Examples(`
		jx step collect TODO
`)
)

func NewCmdStepCollect(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepCollectOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "collect",
		Short:   "Collects the specified files that need storing from the build",
		Long:    StepCollectLong,
		Example: StepCollectExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Provider, "provider", "", "", fmt.Sprintf("Specify the storage provider to use. Supported options are: %s", CollectProvidersKinds))
	cmd.Flags().StringArrayVarP(&options.Pattern, "pattern", "", make([]string, 0), fmt.Sprintf("Specify the pattern to use to look for files"))
	cmd.Flags().StringVarP(&options.HttpStepCollectOptions.Destination, "destination", "", "", fmt.Sprintf("Specify the HTTP endpoint to send each file to"))
	cmd.Flags().StringVarP(&options.Classifier, "classifier", "", "", "A name which classifies this type of file. Example values: " + kube.ClassificationValues)
	return cmd
}

func (o *StepCollectOptions) Run() error {
	if o.Provider == "" {
		return errors.New("Must specify a provider using --provider")
	}
	if strings.ToLower(o.Provider) == strings.ToLower(string(GitHubPagesCollectProviderKind)) {
		err := o.GitHubPagesStepCollectOptions.collect(*o)
		return err
	} else if strings.ToLower(o.Provider) == strings.ToLower(string(HttpCollectProviderKind)) {
		return o.HttpStepCollectOptions.collect()
	} else {
		return errors.New(fmt.Sprintf("Unrecognized provider %s", o.Provider))
	}
	return nil
}

func (o *GitHubPagesStepCollectOptions) collect(options StepCollectOptions) (err error) {
	// Can't assume we are in a git repo due to shallow clones etc.

	sourceURL := os.Getenv(envVarSourceUrl)
	sourceURLParts := regexp.MustCompile(gitHubRepoPattern).FindStringSubmatch(sourceURL)

	if len(sourceURLParts) != 3 {
		return errors.New(fmt.Sprintf("Git repo must be GitHub to use GitHub Pages but it is %s", sourceURL))
	}

	org := sourceURLParts[1]
	repoName := sourceURLParts[2]

	gitClient := options.Git()

	ghPagesDir, err := cloneGitHubPagesBranchToTempDir(sourceURL, gitClient)
	if err != nil {
	  return err
	}
	
	buildNo := options.getBuildNumber()
	if options.Classifier == "" {
		return errors.New("You must pass --classfier")
	}

	branchName := os.Getenv(envVarBranchName)
	if branchName == "" {
		return fmt.Errorf("Environment variable %s is empty", envVarBranchName)
	}

	repoPath := filepath.Join("jenkins-x", options.Classifier, org, repoName, branchName, buildNo)
	repoDir := filepath.Join(ghPagesDir, repoPath)
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	for _, p := range options.Pattern {
		_, err = exec.Command("cp", "-r", p, repoDir).Output()
		if err != nil {
			return err
		}
	}

	urls := make([]string, 0)
	err = filepath.Walk(repoDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				rPath := strings.TrimPrefix(strings.TrimPrefix(path, ghPagesDir), "/")

				if rPath != "" {
					url := fmt.Sprintf("https://%s.github.io/%s/%s", org, repoName, rPath)
					log.Infof("Publishing %s\n", util.ColorInfo(url))
					urls = append(urls, url)
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = gitClient.Add(ghPagesDir, repoDir)
	if err != nil {
		return err
	}
	err = gitClient.CommitDir(ghPagesDir, fmt.Sprintf("Publishing files for build %s", buildNo))
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = gitClient.Push(ghPagesDir)
	if err != nil {
		return err
	}

	f := options.Factory
	client, ns, err := f.CreateJXClient()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
	}

	apisClient, err := options.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	activities := client.JenkinsV1().PipelineActivities(ns)
	if err != nil {
		return err
	}
	build := options.getBuildNumber()
	// TODO this pipeline name construction needs moving to a shared lib, and other things refactoring to use it
	pipeline := fmt.Sprintf("%s-%s-%s-%s", org, repoName, branchName, build)

	if pipeline != "" && build != "" {
		name := kube.ToValidName(pipeline)
		key := &kube.PromoteStepActivityKey{
			PipelineActivityKey: kube.PipelineActivityKey{
				Name:     name,
				Pipeline: pipeline,
				Build:    build,
			},
		}
		a, _, err := key.GetOrCreate(activities)
		if err != nil {
			return err
		}
		a.Spec.Attachments = append(a.Spec.Attachments, jenkinsv1.Attachment{
			Name: options.Classifier,
			URLs: urls,
		})
		_, err = client.JenkinsV1().PipelineActivities(ns).Update(a)
		if err != nil {
			return err
		}
	}
	return nil
}

// cloneGitHubPagesBranchToTempDir clones the github pages branch to a temp dir
func cloneGitHubPagesBranchToTempDir(sourceURL string, gitClient gits.Gitter) (string, error) {
	// First clone the git repo
	ghPagesDir, err := ioutil.TempDir("", "jenkins-x-collect")
	if err != nil {
		return ghPagesDir, err
	}

	err = gitClient.ShallowCloneBranch(sourceURL, ghPagesBranchName, ghPagesDir)
	if err != nil {
		log.Infof("error doing shallow clone of gh-pages %v", err)
		// swallow the error
		log.Infof("No existing %s branch\n", ghPagesBranchName)
		// branch doesn't exist, so we create it following the process on https://help.github.com/articles/creating-project-pages-using-the-command-line/
		err = gitClient.Clone(sourceURL, ghPagesDir)
		if err != nil {
			return ghPagesDir, err
		}
		err = gitClient.CheckoutOrphan(ghPagesDir, ghPagesBranchName)
		if err != nil {
			return ghPagesDir, err
		}
		err = gitClient.RemoveForce(ghPagesDir, ".")
		if err != nil {
			return ghPagesDir,  err
		}
		err = os.Remove(filepath.Join(ghPagesDir, ".gitignore"))
		if err != nil {
			// Swallow the error, doesn't matter
		}
	}
	return ghPagesDir, nil
}

func (o *GitHubPagesStepCollectOptions) contains(strings []string, str string) bool {
	for _, s := range strings {
		if str == s {
			return true
		}
	}
	return false
}

func (o *HttpStepCollectOptions) collect() error {
	return nil
}
