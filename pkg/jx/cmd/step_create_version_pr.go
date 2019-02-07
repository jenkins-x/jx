package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	pipelineapi "github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/uuid"
	"strings"
)

var (
	createVersionPullRequestLong = templates.LongDesc(`
		Creates a Pull Request on the versions git repository for a new version of a chart/package
`)

	createVersionPullRequestExample = templates.Examples(`
		# create a Pull Request to update a chart version
		jx step create version pr -n jenkins-x/prow -v 1.2.3

			`)
)

// StepCreateVersionPullRequestOptions contains the command line flags
type StepCreateVersionPullRequestOptions struct {
	StepOptions

	Kind               string
	Name               string
	VersionsRepository string
	VersionsBranch     string
	Version            string
}

// StepCreateVersionPullRequestResults stores the generated results
type StepCreateVersionPullRequestResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreateVersionPullRequest Creates a new Command object
func NewCmdStepCreateVersionPullRequest(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateVersionPullRequestOptions{
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
		Use:     "version pr",
		Short:   "Creates a Pull Request on the versions git repository for a new version of a chart/package",
		Long:    createVersionPullRequestLong,
		Example: createVersionPullRequestExample,
		Aliases: []string{"version pullrequest"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.VersionsRepository, "repo", "r", DefaultVersionsURL, "Jenkins X versions Git repo")
	cmd.Flags().StringVarP(&options.VersionsBranch, "branch", "", "master", "the versions git repository branch to clone and generate a pull request from")
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "charts", "The kind of version such as 'charts' or 'packages'")
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the version to update. e.g. the name of the chart like 'jenkins-x/prow'")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version to change")
	return cmd
}

// Run implements this command
func (o *StepCreateVersionPullRequestOptions) Run() error {
	if o.Name == "" {
		return util.MissingOption("name")
	}
	if o.Kind == "" {
		return util.MissingOption("kind")
	}
	if o.Version == "" {
		return util.MissingOption("version")
	}
	if o.VersionsRepository == "" {
		return util.MissingOption("repo")
	}
	if o.VersionsBranch == "" {
		o.VersionsBranch = "master"
	}

	dir, err := ioutil.TempDir("", "create-version-pr")
	if err != nil {
		return err
	}

	gitInfo, err := gits.ParseGitURL(o.VersionsRepository)
	if err != nil {
		return err
	}
	provider, err := o.gitProviderForURL(o.VersionsRepository, "versions repository")
	if err != nil {
		return err
	}

	username := provider.CurrentUsername()
	if username == "" {
		return fmt.Errorf("no git user name found")
	}

	originalOrg := gitInfo.Organisation
	originalRepo := gitInfo.Name

	repo, err := provider.GetRepository(username, originalRepo)
	if err != nil {
		if originalOrg == username {
			return err
		}

		// lets try create a fork
		repo, err = provider.ForkRepository(originalOrg, originalRepo, username)
		if err != nil {
			return errors.Wrapf(err, "failed to fork GitHub repo %s/%s to user %s", originalOrg, originalRepo, username)
		}
		log.Infof("Forked Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	}

	originalGitURL := o.VersionsRepository
	git := o.Git()

	err = git.Clone(repo.CloneURL, dir)
	if err != nil {
		return errors.Wrapf(err, "cloning the version repository %q", repo.CloneURL)
	}
	log.Infof("cloned fork of version repository %s to %s\n", util.ColorInfo(repo.HTMLURL), util.ColorInfo(dir))

	err = git.SetRemoteURL(dir, "upstream", originalGitURL)
	if err != nil {
		return errors.Wrapf(err, "setting remote upstream %q in forked environment repo", originalGitURL)
	}
	err = git.PullUpstream(dir)
	if err != nil {
		return errors.Wrap(err, "pulling upstream of forked versions repository")
	}

	branchNameText := strings.Replace("upgrade-"+o.Name+"-"+o.Version, "/", "-", -1)
	branchNameText = strings.Replace(branchNameText, ".", "-", -1)

	title := fmt.Sprintf("change %s to version %s", o.Name, o.Version)
	message := fmt.Sprintf("change %s to version %s", o.Name, o.Version)

	branchName := o.Git().ConvertToValidBranchName(branchNameText)
	branchNames, err := o.Git().RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return errors.Wrapf(err, "failed to load remote branch names")
	}
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}

	err = git.CreateBranch(dir, branchName)
	if err != nil {
		return err
	}
	err = git.Checkout(dir, branchName)
	if err != nil {
		return err
	}

	err = o.modifyFiles(dir)
	if err != nil {
		return err
	}

	err = o.Git().Add(dir, "*", "*/*")
	if err != nil {
		return err
	}
	changes, err := git.HasChanges(dir)
	if err != nil {
		return err
	}
	if !changes {
		log.Infof("No source changes so not generating a Pull Request\n")
		return nil
	}

	err = git.CommitDir(dir, title)
	if err != nil {
		return err
	}
	err = git.Push(dir)
	if err != nil {
		return errors.Wrapf(err, "pushing forked environment dir %q", dir)
	}

	base := o.VersionsBranch

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         title,
		Body:          message,
		Base:          base,
		Head:          username + ":" + branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return err
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return nil
}

func (o *StepCreateVersionPullRequestOptions) modifyFiles(dir string) error {
	kind := version.VersionKind(o.Kind)
	data, err := version.LoadStableVersion(dir, kind, o.Name)
	if err != nil {
		return err
	}
	if data.Version == o.Version {
		return nil
	}
	data.Version = o.Version
	err = version.SaveStableVersion(dir, kind, o.Name, data)
	if err != nil {
		return errors.Wrapf(err, "failed to save version file")
	}
	return nil
}
