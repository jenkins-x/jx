package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"

	"github.com/spf13/cobra"
)

const (
	//     * runs (aka 'run')

	valid_resources = `Valid resource types include:

    * environments (aka 'env')
    * pipelines (aka 'pipe')
    * urls (aka 'url')
    `
)

// NewJXCommand creates the `jx` command and its nested children.
func NewJXCommand(f cmdutil.Factory, in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "jx",
		Short: "jx is a command line tool for working with Jenkins X",
		Long: `
 `,
		Run: runHelp,
		/*
			BashCompletionFunction: bash_completion_func,
		*/
	}

	/*
		f.BindFlags(cmds.PersistentFlags())
		f.BindExternalFlags(cmds.PersistentFlags())

		// From this point and forward we get warnings on flags that contain "_" separators
		cmds.SetGlobalNormalizationFunc(flag.WarnWordSepNormalizeFunc)

		groups := templates.CommandGroups{
			{
				Message: "Basic Commands (Beginner):",
				Commands: []*cobra.Command{
					NewCmdCreate(f, out, err),
					NewCmdExposeService(f, out),
					NewCmdRun(f, in, out, err),
					set.NewCmdSet(f, out, err),
				},
			},
	*/

	cmds.AddCommand(NewCmdCompletion(f, out))
	cmds.AddCommand(NewCmdConsole(f, out, err))
	cmds.AddCommand(NewCmdCreate(f, out, err))
	cmds.AddCommand(NewCmdEdit(f, out, err))
	cmds.AddCommand(NewCmdDelete(f, out, err))
	cmds.AddCommand(NewCmdEnvironment(f, out, err))
	cmds.AddCommand(NewCmdGet(f, out, err))
	cmds.AddCommand(NewCmdImport(f, out, err))
	cmds.AddCommand(NewCmdInit(f, out, err))
	cmds.AddCommand(NewCmdInstall(f, out, err))
	cmds.AddCommand(NewCmdLogs(f, out, err))
	cmds.AddCommand(NewCmdNamespace(f, out, err))
	cmds.AddCommand(NewCmdOpen(f, out, err))
	cmds.AddCommand(NewCmdPromote(f, out, err))
	cmds.AddCommand(NewCmdPrompt(f, out, err))
	cmds.AddCommand(NewCmdUninstall(f, out, err))
	cmds.AddCommand(NewCmdVersion(f, out))

	return cmds
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
