package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCPreviewsOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	GCPreviewsLong = templates.LongDesc(`
		Gargabe collect Jenkins X preview environments.  If a pull request is merged or closed the associated preview
		environment will be deleted.

`)

	GCPreviewsExample = templates.Examples(`
		jx garbage collect previews
		jx gc previews
`)
)

// NewCmd s a command object for the "step" command
func NewCmdGCPreviews(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCPreviewsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "previews",
		Short:   "garbage collection for preview environments",
		Long:    GCPreviewsLong,
		Example: GCPreviewsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *GCPreviewsOptions) Run() error {

	log.Warn("this function is not yet implemented\n")

	return nil
}
