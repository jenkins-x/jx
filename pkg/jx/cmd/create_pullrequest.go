package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	createPullRequestLong = templates.LongDesc(`
		Creates a Pull Request in a the git project of the current directory
`)

	createPullRequestExample = templates.Examples(`
		# Create a Pull Request in the current project
		jx create pullrequest -t "my PR title"


		# Create a Pull Request with a title and a body
		jx create pullrequest -t "my PR title" --body "	
		some more
		text
		goes
		here
		""
"
	`)
)

// CreatePullRequestOptions the options for the create spring command
type CreatePullRequestOptions struct {
	CreateOptions

	Dir    string
	Title  string
	Body   string
	Labels []string
	Base   string

	Results CreatePullRequestResults
}

type CreatePullRequestResults struct {
	GitInfo     *gits.GitRepository
	GitProvider gits.GitProvider
	PullRequest *gits.GitPullRequest
}

// NewCmdCreatePullRequest creates a command object for the "create" command
func NewCmdCreatePullRequest(commonOpts *CommonOptions) *cobra.Command {
	options := &CreatePullRequestOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "pullrequest",
		Short:   "Create a Pull Request on the git project for the current directory",
		Aliases: []string{"pr", "pull request"},
		Long:    createPullRequestLong,
		Example: createPullRequestExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory used to detect the Git repository. Defaults to the current directory")
	cmd.Flags().StringVarP(&options.Title, optionTitle, "t", "", "The title of the pullrequest to create")
	cmd.Flags().StringVarP(&options.Body, "body", "", "", "The body of the pullrequest")
	cmd.Flags().StringVarP(&options.Base, "base", "", "master", "The base branch to create the pull request into")
	cmd.Flags().StringArrayVarP(&options.Labels, "label", "l", []string{}, "The labels to add to the pullrequest")

	return cmd
}

// Run implements the command
func (o *CreatePullRequestOptions) Run() error {
	// lets discover the git dir
	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Dir = dir
	}
	gitInfo, provider, _, err := o.createGitProvider(o.Dir)
	if err != nil {
		return err
	}

	o.Results.GitInfo = gitInfo
	o.Results.GitProvider = provider

	branchName, err := o.Git().Branch(o.Dir)
	if err != nil {
		return err
	}

	arguments := &gits.GitPullRequestArguments{
		Base: o.Base,
		Head: branchName,
	}
	err = o.PopulatePullRequest(arguments, gitInfo)
	if err != nil {
		return err
	}

	pr, err := provider.CreatePullRequest(arguments)
	if err != nil {
		return err
	}

	o.Results.PullRequest = pr

	log.Infof("\nCreated PullRequest %s at %s\n", util.ColorInfo(pr.NumberString()), util.ColorInfo(pr.URL))

	return nil
}

func (o *CreatePullRequestOptions) PopulatePullRequest(pullRequest *gits.GitPullRequestArguments, gitInfo *gits.GitRepository) error {
	title := o.Title
	if title == "" {
		if o.BatchMode {
			return util.MissingOption(optionTitle)
		}
		defaultValue, body, err := o.findLastCommitTitle()
		if err != nil {
			log.Warnf("Failed to find last git commit title: %s\n", err)
		}
		if o.Body == "" {
			o.Body = body
		}
		title, err = util.PickValue("PullRequest title:", defaultValue, true, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	body := o.Body
	pullRequest.Title = title
	pullRequest.Body = body
	pullRequest.GitRepository = gitInfo

	if title == "" {
		return fmt.Errorf("No title specified!")
	}
	return nil
}

func (o *CreatePullRequestOptions) findLastCommitTitle() (string, string, error) {
	title := ""
	body := ""
	dir := o.Dir
	gitDir, gitConfDir, err := o.Git().FindGitConfigDir(dir)
	if err != nil {
		return title, body, err
	}
	if gitDir == "" || gitConfDir == "" {
		log.Warnf("No git directory could be found from dir %s\n", dir)
		return title, body, err
	}
	message, err := o.Git().GetLatestCommitMessage(dir)
	if err != nil {
		return title, body, err
	}
	lines := strings.SplitN(message, "\n", 2)
	if len(lines) < 2 {
		return message, "", nil
	}
	return lines[0], lines[1], nil
}
