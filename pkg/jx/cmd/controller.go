package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// ControllerOptions contains the CLI options
type ControllerOptions struct {
	CommonOptions
}

var (
	controllerLong = templates.LongDesc(`
		Runs a controller

`)

	controllerExample = templates.Examples(`
	`)
)

// NewCmdController creates the edit command
func NewCmdController(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ControllerOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "controller [flags]",
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

	cmd.AddCommand(NewCmdControllerBuild(f, out, errOut))
	cmd.AddCommand(NewCmdControllerWorkflow(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *ControllerOptions) Run() error {
	return o.Cmd.Help()
}
