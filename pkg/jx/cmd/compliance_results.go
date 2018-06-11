package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

var (
	complianceResultsLong = templates.LongDesc(`
		Shows the results of the compliance tests
	`)

	complianceResultsExample = templates.Examples(`
		# Show the compliance results
		jx compliance results
	`)
)

// ComplianceStatusOptions options for "compliance results" command
type ComplianceResultsOptions struct {
	CommonOptions
}

// NewCmdComplianceResults creates a command object for the "compliance results" action, which
// shows the results of E2E compliance tests
func NewCmdComplianceResults(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ComplianceResultsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "results",
		Short:   "Shows the results of compliance tests",
		Long:    complianceResultsLong,
		Example: complianceResultsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the "compliance results" command
func (o *ComplianceResultsOptions) Run() error {
	return o.Cmd.Help()
}
