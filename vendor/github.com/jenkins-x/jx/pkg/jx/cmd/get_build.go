package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetBuildOptions the command line options
type GetBuildOptions struct {
	CommonOptions

	Output string
}

var (
	get_build_long = templates.LongDesc(`
		Display one or many resources.

		` + valid_resources + `

`)

	get_build_example = templates.Examples(`
		# List all pipeines
		jx get pipeline

		# List all URLs for services in the current namespace
		jx get url
	`)
)

// NewCmdGetBuild creates the command object
func NewCmdGetBuild(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetBuildOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "build [flags]",
		Short:   "Display one or many build resources",
		Long:    get_build_long,
		Example: get_build_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdGetBuildLogs(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *GetBuildOptions) Run() error {
	return o.Cmd.Help()
}
