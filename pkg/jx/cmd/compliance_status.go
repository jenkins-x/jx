package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

var (
	complianceStatusLong = templates.LongDesc(`
		Retrieves the current status of the compliance E2E tests
	`)

	complianceStatusExample = templates.Examples(`
		# Get the status
		jx compliance status
	`)
)

// ComplianceStatusOptions options for "compliance status" command
type ComplianceStatusOptions struct {
	CommonOptions
}

// NewCmdComplianceStatus creates a command object for the "compliance status" action, which
// retrieve the status of E2E compliance tests
func NewCmdComplianceStatus(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ComplianceStartOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Retrieve the status of compliance E2E tests",
		Long:    complianceStatusLong,
		Example: complianceStatusExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the "compliance status" command
func (o *ComplianceStatusOptions) Run() error {
	return o.Cmd.Help()
}
