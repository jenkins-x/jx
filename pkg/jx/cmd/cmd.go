package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"

	"github.com/spf13/cobra"
)

const (
	//     * runs (aka 'run')

	valid_resources = `Valid resource types include:

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

	cmds.AddCommand(NewCmdConsole(f, out, err))
	cmds.AddCommand(NewCmdGet(f, out, err))
	cmds.AddCommand(NewCmdImport(f, out, err))
	cmds.AddCommand(NewCmdInstall(f, out, err))
	cmds.AddCommand(NewCmdOpen(f, out, err))
	cmds.AddCommand(NewCmdVersion(f, out))

	return cmds
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
