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
type GCActivitiesOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	GCActivitiesLong = templates.LongDesc(`
		Gargabe collect the Jenkins X Activity Custom Resource Definitions

`)

	GCActivitiesExample = templates.Examples(`
		jx garbage collect activities
		jx gc activities
`)
)

// NewCmd s a command object for the "step" command
func NewCmdGCActivities(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCActivitiesOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "activities",
		Short:   "garbage collection for activities",
		Long:    GCActivitiesLong,
		Example: GCActivitiesExample,
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
func (o *GCActivitiesOptions) Run() error {

	log.Warn("this function is not yet implemented\n")

	return nil
}
