package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"strconv"

	"fmt"

	"github.com/jenkins-x/jx/pkg/gits"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepPRCommentOptions struct {
	StepPROptions
	Flags StepPRCommentFlags
}

type StepPRCommentFlags struct {
	Comment    string
	URL        string
	Owner      string
	Repository string
	PR         string
}

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPRComment(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPRCommentOptions{
		StepPROptions: StepPROptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "comment",
		Short: "pipeline step pr comment",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Flags.Comment, "comment", "c", "", "comment to add to the Pull Request")
	cmd.Flags().StringVarP(&options.Flags.Owner, "owner", "o", "", "Git organisation / owner")
	cmd.Flags().StringVarP(&options.Flags.Repository, "repository", "r", "", "Git repository")
	cmd.Flags().StringVarP(&options.Flags.PR, "pull-request", "p", "", "Git Pull Request number")

	options.addCommonFlags(cmd)

	return cmd
}

// Run implements this command
func (o *StepPRCommentOptions) Run() error {
	if o.Flags.PR == "" {
		return fmt.Errorf("no Pull Request number provided")
	}
	if o.Flags.Owner == "" {
		return fmt.Errorf("no Git owner provided")
	}
	if o.Flags.Repository == "" {
		return fmt.Errorf("no Git repository provided")
	}
	if o.Flags.Comment == "" {
		return fmt.Errorf("no comment provided")
	}

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	gitInfo, err := o.Git().Info("")
	if err != nil {
		return err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return err
	}

	provider, err := o.CreateGitProvider(gitInfo.URL, "user name to submit comment as", authConfigSvc, gitKind, o.BatchMode, o.Git(), o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	prNumber, err := strconv.Atoi(o.Flags.PR)
	if err != nil {
		return err
	}

	pr := gits.GitPullRequest{
		Repo:   o.Flags.Repository,
		Owner:  o.Flags.Owner,
		Number: &prNumber,
	}

	return provider.AddPRComment(&pr, o.Flags.Comment)
}
