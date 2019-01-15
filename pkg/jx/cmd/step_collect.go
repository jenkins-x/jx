package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/collector"
	"io"
	"os"
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
	Pattern         []string
	Dir             string
	Basedir         string
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

		# lets collect some files with the file names relative to the 'target/test-reports' folder and store in a Git URL
		jx step collect -c tests -p "target/test-reports/*" --basedir target/test-reports --git-url https://github.com/myuser/myrepo.git

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
	cmd.Flags().StringArrayVarP(&options.Pattern, "pattern", "p", nil, "Specify the pattern to use to look for files")
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory to try detect the current git repository or branch. Defaults to using the current directory")
	cmd.Flags().StringVarP(&options.Basedir, "basedir", "", "", "The base directory to use to create relative output file names. e.g. if you specify '--pattern \"target/*.xml\" then you may want to supply '--basedir target' to strip the 'target/' prefix from all collected files")
	cmd.Flags().StringVarP(&options.StorageLocation.HttpURL, "http-url", "", "", "Specify the HTTP endpoint to send each file to")
	cmd.Flags().StringVarP(&options.StorageLocation.GitURL, "git-url", "", "", "Specify the Git URL to populate files in a gh-pages branch")
	cmd.Flags().StringVarP(&options.StorageLocation.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	return cmd
}

func (o *StepCollectOptions) Run() error {
	if len(o.Pattern) == 0 {
		return util.MissingOption("pattern")
	}
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
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	if o.StorageLocation.IsEmpty() {
		// lets try get the location from the team settings
		o.StorageLocation = *settings.StorageLocationOrDefault(classifier)

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
	if o.StorageLocation.IsEmpty() {
		return fmt.Errorf("Missing option --git-url and we could not detect the current git repository URL")
	}

	coll, err := collector.NewCollector(&o.StorageLocation, settings, o.Git())
	if err != nil {
		return err
	}

	client, ns, err := o.CreateJXClient()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
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

	repoPath := filepath.Join("jenkins-x", classifier, projectOrg, projectRepoName, projectBranchName, buildNo)

	urls, err := coll.CollectFiles(o.Pattern, repoPath, o.Basedir)
	if err != nil {
		return errors.Wrapf(err, "failed to collect patterns %s to path %s", strings.Join(o.Pattern, ", "), repoPath)
	}

	// TODO this pipeline name construction needs moving to a shared lib, and other things refactoring to use it
	pipeline := fmt.Sprintf("%s-%s-%s-%s", projectOrg, projectRepoName, projectBranchName, buildNo)
	activities := client.JenkinsV1().PipelineActivities(ns)

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
