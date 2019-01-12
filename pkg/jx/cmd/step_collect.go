package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"

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
	Pattern         []string
	Dir             string
	StorageLocation jenkinsv1.StorageLocation
}

const (
	envVarBranchName = "BRANCH_NAME"
	envVarSourceUrl  = "SOURCE_URL"
)

const ghPagesBranchName = "gh-pages"

var (
	StepCollectLong = templates.LongDesc(`
		This pipeline step collects the specified files that need storing from the build into some stable storage location 
`)

	StepCollectExample = templates.Examples(`
		# lets collect some files to the team's default storage location (which if not specified using the current git repository's gh-pages branch)
		jx step collect -c tests -p "target/test-reports/*"

		# lets collect some files to a specific Git URL
		jx step collect -c tests -p "target/test-reports/*" --git-url https://github.com/myuser/myrepo.git

		# lets collect some files to a specific HTTP URL
		jx step collect -c coverage -p "build/coverage/*" --http-url https://myserver.cheese/

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
	cmd.Flags().StringArrayVarP(&options.Pattern, "pattern", "p", make([]string, 0), "Specify the pattern to use to look for files")
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory to try detect the current git repository or branch. Defaults to using the current directory")
	cmd.Flags().StringVarP(&options.StorageLocation.HttpURL, "http-url", "", "", "Specify the HTTP endpoint to send each file to")
	cmd.Flags().StringVarP(&options.StorageLocation.GitURL, "git-url", "", "", "Specify the Git URL to populate files in a gh-pages branch")
	cmd.Flags().StringVarP(&options.StorageLocation.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	return cmd
}

func (o *StepCollectOptions) Run() error {
	classifier := o.StorageLocation.Classifier
	if classifier == "" {
		return util.MissingOption("classifier")
	}
	var err error
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if o.StorageLocation.IsEmpty() {
		// lets try get the location from the team settings
		settings, err := o.TeamSettings()
		if err != nil {
			return err
		}
		o.StorageLocation = *settings.StorageLocation(classifier)

		if o.StorageLocation.IsEmpty() {
			// we have no team settings so lets try detect the git repository using an env var or local file system
			sourceURL := os.Getenv(envVarSourceUrl)
			if sourceURL == "" {
				_, gitConf, err := o.Git().FindGitConfigDir(o.Dir)
				if err != nil {
					log.Warnf("Could not find a .git directory: %s\n", err)
				} else {
					sourceURL, err = o.discoverGitURL(gitConf)
				}
			}
			if sourceURL == "" {
				return fmt.Errorf("Missing option --git-url and we could not detect the current git repository URL")
			}
			o.StorageLocation.GitURL = sourceURL
		}
	}

	gitURL := o.StorageLocation.GitURL
	if gitURL != "" {
		return o.collectGitURL(gitURL)
	}
	httpURL := o.StorageLocation.HttpURL
	if httpURL != "" {
		return o.collectHttpURL(httpURL)
	}
	return fmt.Errorf("Missing option --git-url and we could not detect the current git repository URL")
}

func (o *StepCollectOptions) collectGitURL(storageURL string) (err error) {
	storageGitInfo, err := gits.ParseGitURL(storageURL)
	if err != nil {
		return err
	}
	storageOrg := storageGitInfo.Organisation
	storageRepoName := storageGitInfo.Name

	gitClient := o.Git()

	ghPagesDir, err := cloneGitHubPagesBranchToTempDir(storageURL, gitClient)
	if err != nil {
		return err
	}

	buildNo := o.getBuildNumber()
	projectGitInfo, err := o.FindGitInfo("")
	if err != nil {
		return err
	}
	projectOrg := projectGitInfo.Organisation
	projectRepoName := projectGitInfo.Name

	projectBranchName := os.Getenv(envVarBranchName)
	if projectBranchName == "" {
		// lets try find the branch name via git
		projectBranchName, err = o.Git().Branch(o.Dir)
		if err != nil {
			return err
		}
	}
	if projectBranchName == "" {
		return fmt.Errorf("Environment variable %s is empty", envVarBranchName)
	}

	classifier := o.StorageLocation.Classifier
	repoPath := filepath.Join("jenkins-x", classifier, projectOrg, projectRepoName, projectBranchName, buildNo)
	repoDir := filepath.Join(ghPagesDir, repoPath)
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	for _, p := range o.Pattern {
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
					url := fmt.Sprintf("https://%s.github.io/%s/%s", storageOrg, storageRepoName, rPath)
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

	client, ns, err := o.CreateJXClient()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
	}

	apisClient, err := o.ApiExtensionsClient()
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

	// TODO this pipeline name construction needs moving to a shared lib, and other things refactoring to use it
	pipeline := fmt.Sprintf("%s-%s-%s-%s", projectOrg, projectRepoName, projectBranchName, buildNo)

	if pipeline != "" && buildNo != "" {
		name := kube.ToValidName(pipeline)
		key := &kube.PromoteStepActivityKey{
			PipelineActivityKey: kube.PipelineActivityKey{
				Name:     name,
				Pipeline: pipeline,
				Build:    buildNo,
			},
		}
		a, _, err := key.GetOrCreate(activities)
		if err != nil {
			return err
		}
		a.Spec.Attachments = append(a.Spec.Attachments, jenkinsv1.Attachment{
			Name: classifier,
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
		log.Infof("No existing %s branch so creating it\n", ghPagesBranchName)
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
			return ghPagesDir, err
		}
		err = os.Remove(filepath.Join(ghPagesDir, ".gitignore"))
		if err != nil {
			// Swallow the error, doesn't matter
		}
	}
	return ghPagesDir, nil
}

func (o *StepCollectOptions) collectHttpURL(httpURL string) error {
	return fmt.Errorf("TODO! Not implemented yet! Cannot post to %s\n", httpURL)
}
