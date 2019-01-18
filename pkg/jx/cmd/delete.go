package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// DeleteOptions are the flags for delete commands
type DeleteOptions struct {
	CommonOptions
}

var (
	delete_long = templates.LongDesc(`
		Deletes one or more resources.

`)

	delete_example = templates.Examples(`
		# Delete an environment
		jx delete env staging
	`)
)

// NewCmdDelete creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDelete(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteOptions{
		CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "delete TYPE [flags]",
		Short:   "Deletes one or more resources",
		Long:    delete_long,
		Example: delete_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm", "del"},
	}

	cmd.AddCommand(NewCmdDeleteAddon(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteApplication(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteBranch(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteChat(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteContext(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteDevPod(f, in, out, errOut))
	cmd.AddCommand(newCmdDeleteEks(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteGit(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteJenkins(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteNamespace(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeletePostPreviewJob(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeletePreview(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteQuickstartLocation(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteRepo(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteToken(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteTeam(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteTracker(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteUser(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteAws(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteVault(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteExtension(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteOptions) Run() error {
	return o.Cmd.Help()
}
