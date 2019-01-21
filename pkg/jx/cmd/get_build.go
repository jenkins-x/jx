package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// GetBuildOptions the command line options
type GetBuildOptions struct {
	commoncmd.CommonOptions

	Output string
}

var (
	get_build_long = templates.LongDesc(`
		Display one or more resources.

		` + valid_resources + `

`)

	get_build_example = templates.Examples(`
		# List all pipelines
		jx get pipeline

		# List all URLs for services in the current namespace
		jx get url
	`)
)

// NewCmdGetBuild creates the command object
func NewCmdGetBuild(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetBuildOptions{
		CommonOptions: commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "build [flags]",
		Short:   "Display one or more build resources",
		Long:    get_build_long,
		Example: get_build_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdGetBuildLogs(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetBuildPods(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *GetBuildOptions) Run() error {
	return o.Cmd.Help()
}
