package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteOptions are the flags for delete commands
type DeleteOptions struct {
	CommonOptions
}

var (
	delete_long = templates.LongDesc(`
		Deletes one or many resources.

`)

	delete_example = templates.Examples(`
		# Delete an environment
		jx delete env staging
	`)
)

// NewCmdDelete creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDelete(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "delete TYPE [flags]",
		Short:   "Deletes one or many resources",
		Long:    delete_long,
		Example: delete_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm", "del"},
	}

	cmd.AddCommand(NewCmdDeleteAddon(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteApp(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteChat(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteContext(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteDevPod(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteEnv(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteGit(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteJenkins(f, out, errOut))
	cmd.AddCommand(NewCmdDeletePreview(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteQuickstartLocation(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteRepo(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteToken(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteTracker(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteOptions) Run() error {
	return o.Cmd.Help()
}
