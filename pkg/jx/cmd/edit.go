package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// EditOptions contains the CLI options
type EditOptions struct {
	CommonOptions
}

var (
	exit_long = templates.LongDesc(`
		Edit a resource

`)

	exit_example = templates.Examples(`
		# Lets edit the staging Environment
		jx edit env staging
	`)
)

// NewCmdEdit creates the edit command
func NewCmdEdit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &EditOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "edit [flags]",
		Short:   "Edit a resource",
		Long:    exit_long,
		Example: exit_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdEditAddon(f, out, errOut))
	cmd.AddCommand(NewCmdCreateBranchPattern(f, out, errOut))
	cmd.AddCommand(NewCmdEditBuildpack(f, out, errOut))
	cmd.AddCommand(NewCmdEditConfig(f, out, errOut))
	cmd.AddCommand(NewCmdEditEnv(f, out, errOut))
	cmd.AddCommand(NewCmdEditHelmBin(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *EditOptions) Run() error {
	return o.Cmd.Help()
}
