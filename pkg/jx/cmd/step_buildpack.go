package cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

// StepBuildPackOptions contains the command line flags
type StepBuildPackOptions struct {
	StepOptions
}

// NewCmdStepBuildPack Steps a command object for the "step" command
func NewCmdStepBuildPack(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepBuildPackOptions{
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
		Use:     "buildpack",
		Short:   "buildpack [command]",
		Aliases: buildPacksAliases,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepBuildPackApply(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepBuildPackOptions) Run() error {
	return o.Cmd.Help()
}
