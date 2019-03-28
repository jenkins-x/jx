package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// ControllerOptions contains the CLI options
type ControllerOptions struct {
	*CommonOptions
}

var (
	controllerLong = templates.LongDesc(`
		Runs a controller

`)

	controllerExample = templates.Examples(`
	`)
)

// NewCmdController creates the edit command
func NewCmdController(commonOpts *CommonOptions) *cobra.Command {
	options := &ControllerOptions{
		commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "controller <command> [flags]",
		Short:   "Runs a controller",
		Long:    controllerLong,
		Example: controllerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdControllerBackup(commonOpts))
	cmd.AddCommand(NewCmdControllerBuild(commonOpts))
	cmd.AddCommand(NewCmdControllerBuildNumbers(commonOpts))
	cmd.AddCommand(NewCmdControllerPipelineRunner(commonOpts))
	cmd.AddCommand(NewCmdControllerRole(commonOpts))
	cmd.AddCommand(NewCmdControllerTeam(commonOpts))
	cmd.AddCommand(NewCmdControllerWorkflow(commonOpts))
	cmd.AddCommand(NewCmdControllerCommitStatus(commonOpts))
	return cmd
}

// Run implements this command
func (o *ControllerOptions) Run() error {
	return o.Cmd.Help()
}
