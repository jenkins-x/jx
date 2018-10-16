package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd.Flags().StringVarP(&options.Classifier, "classifier", "", "", "A name which classifies this type of file e.g. test-reports, coverage")
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
	gitRepoInfo, err := options.FindGitInfo("")
	if err != nil {
		return err
	}
	if !gitRepoInfo.IsGitHub() {
		return errors.New(fmt.Sprintf("Git repo must be GitHub to use GitHub Pages but it is %s", gitRepoInfo))
	}

	// First copy files out of the way, as we're going to checkout a different git branch
	collectedDir, err := ioutil.TempDir("", "jenkins-x-collect")
	if err != nil {
		return err
	}
	for _, p := range options.Pattern {
		_, err = exec.Command("cp", "-r", p, collectedDir).Output()
		if err != nil {
			return err
		}
	}

	gitClient := options.Git()
	cwb, err := gitClient.Branch("")
	// TODO duplicated below
	defer func() {
		if gitClient.Checkout("", cwb) != nil {
			log.Errorf("Error checking out original banch %s\n", cwb)
		}
	}()
	if err != nil {
		return err
	}
	err = gitClient.FetchBranch("", "origin", "gh-pages:gh-pages")
	if err != nil {
		// swallow the error
		log.Infof("No existing gh-pages branch")
	}
	remotes, err := gitClient.RemoteBranchNames("", "")
	if err != nil {
		return err
	}
	buildNo := options.getBuildNumber()
	if options.Classifier == "" {
		return errors.New("You must pass --classfier")
	}
	if !contains(remotes, "gh-pages") {
		// branch doesn't exist, so we create it following the process on https://help.github.com/articles/creating-project-pages-using-the-command-line/
		err = gitClient.CheckoutOrphan("", "gh-pages")
		if err != nil {
			return err
		}
		err = gitClient.RemoveForce("", ".")
		if err != nil {
			return err
		}
		err = gitClient.Remove("", ".gitignore")
		if err != nil {
			return err
		}
	}
	err = gitClient.Checkout("", "gh-pages")
	if err != nil {
		return err
	}
	repoDir := filepath.Join("jenkins-x", buildNo, options.Classifier)
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	urls := make([]string, 0)
	err = filepath.Walk(collectedDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rPath := strings.TrimPrefix(strings.TrimPrefix(path, collectedDir), "/")
			rDir := filepath.Dir(rPath)
			err = os.MkdirAll(filepath.Join(repoDir, rDir), 0755)
			if err != nil {
				return err
			}
			if rPath != "" {
				cmd := util.Command{
					Name: "cp",
					Args: []string{path, fmt.Sprintf("%s/%s", repoDir, rPath)},
				}
				_, err := cmd.RunWithoutRetry()
				if err != nil {
					return err
				}
				url := fmt.Sprintf("https://%s.github.com/%s/%s/%s", gitRepoInfo.Organisation, gitRepoInfo.Name, repoDir, rPath)
				log.Infof("Publishing %s\n", util.ColorInfo(url))
				urls = append(urls, url)
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = gitClient.Add("", repoDir)
	if err != nil {
		return err
	}
	err = gitClient.CommitDir("", fmt.Sprintf("Collecting files for build %s", buildNo))
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = gitClient.Push("")
	if err != nil {
		return err
	}

	err = gitClient.Checkout("", cwb)
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
	appName := ""
	if gitRepoInfo != nil {
		appName = gitRepoInfo.Name
	}
	pipeline := ""
	build := options.getBuildNumber()
	pipeline, build = options.getPipelineName(gitRepoInfo, pipeline, build, appName)
	if pipeline != "" && build != "" {
		name := kube.ToValidName(pipeline + "-" + build)
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
