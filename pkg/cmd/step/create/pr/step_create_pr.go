package pr

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/dependencymatrix"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/gits/releases"

	"github.com/jenkins-x/jx/pkg/log"

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
	GitURLs     []string
	Base        string
	Fork        bool
	SrcGitURL   string
	Component   string
	Version     string
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
	cmd.AddCommand(NewCmdStepCreatePullRequestChart(commonOpts))
	cmd.AddCommand(NewCmdStepCreatePullRequestRegex(commonOpts))
	cmd.AddCommand(NewCmdStepCreatePullRequestRepositories(commonOpts))
	cmd.AddCommand(NewCmdStepCreateVersionPullRequest(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepCreatePrOptions) Run() error {
	return o.Cmd.Help()
}

//AddStepCreatePrFlags adds the common flags for all PR creation steps to the cmd and stores them in o
func AddStepCreatePrFlags(cmd *cobra.Command, o *StepCreatePrOptions) {
	cmd.Flags().StringArrayVarP(&o.GitURLs, "repo", "r", []string{}, "Git repo update")
	cmd.Flags().StringVarP(&o.BranchName, "branch", "", "master", "Branch to clone and generate a pull request from")
	cmd.Flags().StringVarP(&o.Base, "base", "", "master", "The branch to create the pull request into")
	cmd.Flags().StringVarP(&o.SrcGitURL, "srcRepo", "", "", "The git repo which caused this change; if this is a dependency update this will cause commit messages to be generated which can be parsed by jx step changelog. By default this will be read from the environment variable REPO_URL")
	cmd.Flags().StringVarP(&o.Component, "component", "", "", "The component of the git repo which caused this change; useful if you have a complex or monorepo setup and want to differentiate between different components from the same repo")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "The version to change. If no version is supplied the latest version is found")

}

// ValidateOptions validates the common options for all PR creation steps
func (o *StepCreatePrOptions) ValidateOptions() error {
	if o.SrcGitURL == "" {
		o.SrcGitURL = os.Getenv("REPO_URL")
		if o.SrcGitURL != "" {
			log.Logger().Infof("Using %s as source for change discovered from env var REPO_URL", o.SrcGitURL)
		} else {
			// see if we're in a git repo and use it
			wd, err := os.Getwd()
			if err != nil {
				return errors.Wrapf(err, "getting working directory")
			}
			gitInfo, err := o.FindGitInfo(wd)
			if err != nil {
				log.Logger().Debugf("Unable to discover git info from current directory because %v", err)
			} else {
				o.SrcGitURL = gitInfo.HttpsURL()
				log.Logger().Infof("Using %s as source for change discovered from git repo in %s", o.SrcGitURL, wd)
			}
		}

	}
	if len(o.GitURLs) == 0 {
		return util.MissingOption("repo")
	}
	return nil
}

// CreatePullRequest will fork (if needed) and pull a git repo, then perform the update, and finally create or update a
// PR for the change. Any open PR on the repo with the `updatebot` label will be updated.
func (o *StepCreatePrOptions) CreatePullRequest(kind string, update func(dir string, gitInfo *gits.GitRepository) ([]string, error)) error {
	for _, gitUrl := range o.GitURLs {
		dir, err := ioutil.TempDir("", "create-pr")
		if err != nil {
			return err
		}

		provider, _, err := o.CreateGitProviderForURLWithoutKind(gitUrl)
		if err != nil {
			return errors.Wrapf(err, "creating git provider for directory %s", dir)
		}

		dir, _, gitInfo, err := gits.ForkAndPullPullRepo(gitUrl, dir, o.Base, o.BranchName, provider, o.Git(), o.ConfigGitFn)
		if err != nil {
			return errors.Wrapf(err, "failed to fork and pull %s", o.GitURLs)
		}

		oldVersions, err := update(dir, gitInfo)
		if err != nil {
			return errors.WithStack(err)
		}

		commitMessage, details, updateDependency, err := o.CreateDependencyUpdatePRDetails(kind, o.SrcGitURL, gitInfo, strings.Join(oldVersions, ", "), o.Version, o.Component)
		if err != nil {
			return errors.WithStack(err)
		}

		err = dependencymatrix.UpdateDependencyMatrix(dir, updateDependency)
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
	}
	return nil
}

// CreateDependencyUpdatePRDetails creates the PullRequestDetails for a pull request, taking the kind of change it is (an id)
// the srcRepoUrl for the repo that caused the change, the destRepo for the repo receiving the change, the fromVersion and the toVersion
func (o *StepCreatePrOptions) CreateDependencyUpdatePRDetails(kind string, srcRepoURL string, destRepo *gits.GitRepository, fromVersion string, toVersion string, component string) (string, *gits.PullRequestDetails, *v1.DependencyUpdate, error) {

	var commitMessage, title, message strings.Builder
	commitMessage.WriteString("chore(dependencies): update ")
	title.WriteString("chore(dependencies): update ")
	message.WriteString("Update ")
	var update *v1.DependencyUpdate

	if srcRepoURL != "" {
		provider, srcRepo, err := o.CreateGitProviderForURLWithoutKind(srcRepoURL)
		if err != nil {
			return "", nil, nil, errors.Wrapf(err, "creating git provider for %s", srcRepoURL)
		}
		update = &v1.DependencyUpdate{
			Owner: srcRepo.Organisation,
			Repo:  srcRepo.Name,
			URL:   srcRepoURL,
		}
		if srcRepo.Host != destRepo.Host {
			commitMessage.WriteString(srcRepoURL)
			title.WriteString(srcRepoURL)
			update.Host = srcRepo.Host
		} else {
			titleStr := fmt.Sprintf("%s/%s", srcRepo.Organisation, srcRepo.Name)
			commitMessage.WriteString(titleStr)
			title.WriteString(titleStr)
			update.Host = destRepo.Host
		}
		repoStr := fmt.Sprintf("[%s/%s](%s)", srcRepo.Organisation, srcRepo.Name, srcRepoURL)
		message.WriteString(repoStr)

		if component != "" {
			componentStr := fmt.Sprintf(":%s", component)
			commitMessage.WriteString(componentStr)
			title.WriteString(componentStr)
			message.WriteString(componentStr)
			update.Component = component
		}
		commitMessage.WriteString(" ")
		title.WriteString(" ")
		message.WriteString(" ")

		if fromVersion != "" {
			fromText := fmt.Sprintf("from %s ", fromVersion)
			commitMessage.WriteString(fromText)
			title.WriteString(fromText)
			update.FromVersion = fromVersion
			release, err := releases.GetRelease(fromVersion, srcRepo.Organisation, srcRepo.Name, provider)
			if err != nil {
				return "", nil, nil, errors.Wrapf(err, "getting release from %s/%s", srcRepo.Organisation, srcRepo.Name)
			}
			if release != nil {
				message.WriteString(fmt.Sprintf("from [%s](%s) ", fromVersion, release.HTMLURL))
				update.FromReleaseName = release.Name
				update.FromReleaseHTMLURL = release.HTMLURL
			} else {
				message.WriteString(fmt.Sprintf("from %s ", fromVersion))
			}
		}
		if toVersion != "" {
			toText := fmt.Sprintf("to %s", toVersion)
			commitMessage.WriteString(toText)
			title.WriteString(toText)
			update.ToVersion = toVersion
			release, err := releases.GetRelease(toVersion, srcRepo.Organisation, srcRepo.Name, provider)
			if err != nil {
				return "", nil, nil, errors.Wrapf(err, "getting release from %s/%s", srcRepo.Organisation, srcRepo.Name)
			}

			if release != nil {
				message.WriteString(fmt.Sprintf("to [%s](%s)", toVersion, release.HTMLURL))
				update.ToReleaseHTMLURL = release.HTMLURL
				update.ToReleaseName = release.Name
			} else {
				message.WriteString(fmt.Sprintf("to %s", toVersion))
			}
		}
	}
	message.WriteString(fmt.Sprintf("\n\nCommand run was `%s`", strings.Join(os.Args, " ")))
	return commitMessage.String(), &gits.PullRequestDetails{
		BranchName: fmt.Sprintf("update-%s-version-%s", kind, string(uuid.NewUUID())),
		Title:      title.String(),
		Message:    message.String(),
	}, update, nil
}
