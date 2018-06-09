package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

var (
	complianceStartLong = templates.LongDesc(`
		Starts the compliance E2E tests
	`)

	complianceStartExample = templates.Examples(`
		# Start the compliance tests
		jx compliance start
	`)
)

// ComplianceStartOptions options for "compliance start" command
type ComplianceStartOptions struct {
	CommonOptions
}

// NewCmdComplianceStart creates a command object for the "compliance start" action, which
// starts the E2E compliance tests
func NewCmdComplianceStart(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ComplianceStartOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "start",
		Short:   "Starts the compliance E2E tests",
		Long:    complianceStartLong,
		Example: complianceStartExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the "compliance start" command
func (o *ComplianceStartOptions) Run() error {
	return o.Cmd.Help()
}
