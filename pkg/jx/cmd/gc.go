package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GCOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCOptions struct {
	CommonOptions

	Output string
}

const (
	valid_gc_resources = `Valid resource types include:

    * previews
    * activities
		* helm
    `
)

var (
	gc_long = templates.LongDesc(`
		Garbage collect resources

		` + valid_gc_resources + `

`)

	gc_example = templates.Examples(`
		jx gc previews
		jx gc activities
		jx gc helm

	`)
)

// NewCmdGC creates a command object for the generic "gc" action, which
// retrieves one or more resources from a server.
func NewCmdGC(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "gc TYPE [flags]",
		Short:   "Garbage collects Jenkins X resources",
		Long:    gc_long,
		Example: gc_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdGCActivities(f, out, errOut))
	cmd.AddCommand(NewCmdGCPreviews(f, out, errOut))
	cmd.AddCommand(NewCmdGCHelm(f, out, errOut))

	return cmd
}

// Run implements this command
func (o *GCOptions) Run() error {
	return o.Cmd.Help()
}
