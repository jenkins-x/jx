package controller

import (
	"github.com/jenkins-x/jx/v2/pkg/cmd/controller/pipeline"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/spf13/cobra"
)

// ControllerOptions contains the CLI options
type ControllerOptions struct {
	*opts.CommonOptions
}

var (
	controllerLong = templates.LongDesc(`
		Runs a controller

`)

	controllerExample = templates.Examples(`
	`)
)

// NewCmdController creates the edit command
func NewCmdController(commonOpts *opts.CommonOptions) *cobra.Command {
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
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdControllerBackup(commonOpts))
	cmd.AddCommand(NewCmdControllerBuild(commonOpts))
	cmd.AddCommand(NewCmdControllerBuildNumbers(commonOpts))
	cmd.AddCommand(NewCmdControllerEnvironment(commonOpts))
	cmd.AddCommand(pipeline.NewCmdControllerPipelineRunner(commonOpts))
	cmd.AddCommand(NewCmdControllerRole(commonOpts))
	cmd.AddCommand(NewCmdControllerTeam(commonOpts))
	cmd.AddCommand(NewCmdControllerCommitStatus(commonOpts))
	return cmd
}

// Run implements this command
func (o *ControllerOptions) Run() error {
	return o.Cmd.Help()
}
