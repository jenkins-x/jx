package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// GetPreviewOptions containers the CLI options
type GetPreviewOptions struct {
	GetEnvOptions
}

var (
	getPreviewLong = templates.LongDesc(`
		Display one or many environments.
`)

	getPreviewExample = templates.Examples(`
		# List all environments
		jx get environments

		# List all environments using the shorter alias
		jx get env
	`)
)

// NewCmdGetPreview creates the new command for: jx get env
func NewCmdGetPreview(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetPreviewOptions{
		GetEnvOptions: GetEnvOptions{
			GetOptions: GetOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "previews",
		Short:   "Display one or many Preview Environments",
		Aliases: []string{"preview"},
		Long:    getPreviewLong,
		Example: getPreviewExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetPreviewOptions) Run() error {
	o.PreviewOnly = true
	return o.GetEnvOptions.Run()
}
