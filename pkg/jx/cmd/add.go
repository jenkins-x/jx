package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// AddOptions contains the command line options
type AddOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	add_resources = `Valid resource types include:

	* app
    `

	add_long = templates.LongDesc(`
		Adds a new resource.

		` + add_resources + `
`)
)

// NewCmdAdd creates a command object for the "add" command
func NewCmdAdd(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &AddOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a new resource",
		Long:  add_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdAddApp(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *AddOptions) Run() error {
	return o.Cmd.Help()
}
