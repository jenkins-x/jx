package pr

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/uuid"
)

//StepCreatePrOptions are the common options for all PR creation steps
type StepCreatePrOptions struct {
	opts.StepCreateOptions
	Results     *gits.PullRequestInfo
	ConfigGitFn gits.ConfigureGitFn
	BranchName  string
	GitURL      string
	Base        string
	Fork        bool
	SrcGitURL   string
}

// NewCmdStepCreatePr Steps a command object for the "step" command
func NewCmdStepCreatePr(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePrOptions{
		StepCreateOptions: opts.StepCreateOptions{
			StepOptions: opts.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pullrequest",
		Aliases: []string{"pr"},
		Short:   "create pullrequest [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepCreatePullRequestDocker(commonOpts))
	cmd.AddCommand(NewCmdStepCreateVersionPullRequest(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepCreatePrOptions) Run() error {
	return o.Cmd.Help()
}

//AddStepCreatePrFlags adds the common flags for all PR creation steps to the cmd and stores them in o
func AddStepCreatePrFlags(cmd *cobra.Command, o *StepCreatePrOptions) {
	cmd.Flags().StringVarP(&o.GitURL, "repo", "r", "", "Git repo")
	cmd.Flags().StringVarP(&o.BranchName, "branch", "", "master", "Branch to clone and generate a pull request from")
	cmd.Flags().StringVarP(&o.Base, "base", "", "master", "The branch to create the pull request into")
	cmd.Flags().StringVarP(&o.SrcGitURL, "srcRepo", "", "", "The git repo which caused this change; if this is a dependency update this will cause commit messages to be generated which can be parsed by jx step changelog. By default this will be read from the environment variable REPO_URL")
}

// ValidateOptions validates the common options for all PR creation steps
func (o *StepCreatePrOptions) ValidateOptions() error {
	if o.GitURL == "" {
		return util.MissingOption("repo")
	}
	return nil
}

// CreatePullRequest will fork (if needed) and pull a git repo, then perform the update, and finally create or update a
// PR for the change. Any open PR on the repo with the `updatebot` label will be updated.
func (o *StepCreatePrOptions) CreatePullRequest(update func(dir string, gitInfo *gits.GitRepository) (string, *gits.PullRequestDetails, error)) error {
	dir, err := ioutil.TempDir("", "create-pr")
	if err != nil {
		return err
	}

	provider, _, err := o.CreateGitProviderForURLWithoutKind(o.GitURL)
	if err != nil {
		return errors.Wrapf(err, "creating git provider for directory %s", dir)
	}

	dir, _, gitInfo, err := gits.ForkAndPullPullRepo(o.GitURL, dir, o.Base, o.BranchName, provider, o.Git(), o.ConfigGitFn)
	if err != nil {
		return errors.Wrapf(err, "failed to fork and pull %s", o.GitURL)
	}

	commitMessage, details, err := update(dir, gitInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	filter := &gits.PullRequestFilter{
		Labels: []string{
			"updatebot",
		},
	}
	o.Results, err = gits.PushRepoAndCreatePullRequest(dir, gitInfo, o.Base, details, filter, true, commitMessage, true, true, provider, o.Git())
	if err != nil {
		return errors.Wrapf(err, "failed to create PR for base %s and head branch %s", o.Base, details.BranchName)
	}
	return nil
}

// CreateDependencyUpdatePRDetails creates the PullRequestDetails for a pull request, taking the kind of change it is (an id)
// the srcRepoUrl for the repo that caused the change, the destRepo for the repo receiving the change, the fromVersion and the toVersion
func CreateDependencyUpdatePRDetails(kind string, srcRepoUrl string, destRepo *gits.GitRepository, fromVersion string, toVersion string) (string, *gits.PullRequestDetails, error) {
	prDetails := gits.PullRequestDetails{}
	prDetails.BranchName = fmt.Sprintf("update-%s-version-%s", kind, string(uuid.NewUUID()))
	fromText := ""
	if fromVersion != "" {
		fromText = fmt.Sprintf("from %s ", fromVersion)
	}

	commitRepo := ""
	repoText := ""
	if srcRepoUrl != "" {
		srcRepo, err := gits.ParseGitURL(srcRepoUrl)
		if err != nil {
			return "", nil, errors.Wrapf(err, "parsing %s as git url", srcRepoUrl)
		}
		if srcRepo.Host != destRepo.Host {
			commitRepo = fmt.Sprintf("%s ", srcRepoUrl)
		} else {
			commitRepo = fmt.Sprintf("%s/%s ", srcRepo.Organisation, srcRepo.Name)
		}

		repoText = fmt.Sprintf("%s/%s ", srcRepo.Organisation, srcRepo.Name)
	}
	commitMessage := fmt.Sprintf("chore(dependencies): update %s%sto %s", commitRepo, fromText, toVersion)
	prDetails.Title = fmt.Sprintf("chore(dependencies): update %s%sto %s", repoText, fromText, toVersion)
	prDetails.Message = fmt.Sprintf("Update %s%sto %s.\n\nCommand run was `%s`", repoText, fromText, toVersion, strings.Join(os.Args, " "))
	return commitMessage, &prDetails, nil
}
