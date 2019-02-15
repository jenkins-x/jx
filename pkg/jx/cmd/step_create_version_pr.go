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

		# create a Pull Request to update a chart version to the latest found in the helm repo
		jx step create version pr -n jenkins-x/prow 

		# create a Pull Request to update all charts matching a filter to the latest found in the helm repo
		jx step create version pr -f "*"

		# create a Pull Request to update all charts in the 'jenkins-x' chart repository to the latest found in the helm repo
		jx step create version pr -f "jenkins-x/*"

			`)
)

// StepCreateVersionPullRequestOptions contains the command line flags
type StepCreateVersionPullRequestOptions struct {
	StepOptions

	Kind               string
	Name               string
	Includes           []string
	Excludes           []string
	VersionsRepository string
	VersionsBranch     string
	Version            string

	updatedHelmRepo bool
	branchNameText  string
	title           string
	message         string
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
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "charts", "The kind of version. Possible values: "+strings.Join(version.KindStrings, ", "))
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the version to update. e.g. the name of the chart like 'jenkins-x/prow'")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version to change. If no version is supplied the latest version is found")
	cmd.Flags().StringArrayVarP(&options.Includes, "filter", "f", nil, "The name patterns to include - such as '*' for all names")
	cmd.Flags().StringArrayVarP(&options.Excludes, "excludes", "x", nil, "The name patterns to exclude")
	return cmd
}

// Run implements this command
func (o *StepCreateVersionPullRequestOptions) Run() error {
	if o.Kind == "" {
		return util.MissingOption("kind")
	}
	if util.StringArrayIndex(version.KindStrings, o.Kind) < 0 {
		return util.InvalidOption("kind", o.Kind, version.KindStrings)
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

	if len(o.Includes) == 0 {
		if o.Name == "" {
			return util.MissingOption("name")
		}

		if o.Version == "" && o.Kind == string(version.KindChart) {
			o.Version, err = o.findLatestChartVersion(o.Name)
			if err != nil {
				return errors.Wrapf(err, "failed to find latest chart version for %s", o.Name)
			}
			log.Infof("found latest version %s for chart %s\n", util.ColorInfo(o.Version), util.ColorInfo(o.Name))
		}
		if o.Version == "" {
			return util.MissingOption("version")
		}
		o.branchNameText = strings.Replace("upgrade-"+o.Name+"-"+o.Version, "/", "-", -1)
		o.branchNameText = strings.Replace(o.branchNameText, ".", "-", -1)

		o.title = fmt.Sprintf("change %s to version %s", o.Name, o.Version)
		o.message = fmt.Sprintf("change %s to version %s", o.Name, o.Version)
	} else {
		o.branchNameText = "upgrade-chart-versions"
		o.title = "upgrade chart versions"
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

		// lets try create a fork - using a blank organisation to force a user specific fork
		repo, err = provider.ForkRepository(originalOrg, originalRepo, "")
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

	branchName := o.Git().ConvertToValidBranchName(o.branchNameText)
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

	err = git.CommitDir(dir, o.title)
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
		Title:         o.title,
		Body:          o.message,
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
	if len(o.Includes) > 0 {
		switch version.VersionKind(o.Kind) {
		case version.KindChart:
			return o.findLatestChartVersions(dir)
		default:
			return fmt.Errorf("We do not yet support finding the latest version of kind %s", o.Kind)
		}
	}

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

func (o *StepCreateVersionPullRequestOptions) findLatestChartVersions(dir string) error {
	callback := func(kind version.VersionKind, name string, stableVersion *version.StableVersion) (bool, error) {
		if !util.StringMatchesAny(name, o.Includes, o.Excludes) {
			return true, nil
		}
		v, err := o.findLatestChartVersion(name)
		if err != nil {
			log.Warnf("failed to find latest version of %s: %s\n", name, err.Error())
			return true, nil
		}
		if v != stableVersion.Version {
			stableVersion.Version = v
			err = version.SaveStableVersion(dir, kind, name, stableVersion)
			if err != nil {
				return false, err
			}
			o.message += fmt.Sprintf("change `%s` to version `%s`\n", name, v)
		}
		return true, nil
	}

	err := version.ForEachKindVersion(dir, version.VersionKind(o.Kind), callback)
	return err
}

func (o *StepCreateVersionPullRequestOptions) findLatestChartVersion(name string) (string, error) {
	err := o.updateHelmRepo()
	if err != nil {
		return "", err
	}
	info, err := o.Helm().SearchChartVersions(name)
	if err != nil {
		return "", err
	}
	if len(info) == 0 {
		return "", fmt.Errorf("no version found for chart %s", name)
	}
	if o.Verbose {
		log.Infof("found %d versions:\n", len(info))
		for _, v := range info {
			log.Infof("    %s:\n", v)
		}
	}
	return info[0], nil
}

// updateHelmRepo updates the helm repos if required
func (o *StepCreateVersionPullRequestOptions) updateHelmRepo() error {
	if o.updatedHelmRepo {
		return nil
	}
	log.Info("updating helm repositories to find the latest chart versions...\n")
	err := o.Helm().UpdateRepo()
	if err != nil {
		return errors.Wrap(err, "failed to update helm repos")
	}
	o.updatedHelmRepo = true
	return nil
}
