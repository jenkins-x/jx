package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

const complianceNamespace = "jx-compliance"

// ComplianceOptions options for compliance command
type ComplianceOptions struct {
	CommonOptions
}

// NewCompliance creates a command object for the generic "compliance" action, which
// executes the compliance tests against a Kubeernetes cluster
func NewCompliance(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ComplianceOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "compliance ACTION [flags]",
		Short: "Run compliance E2E tests against Kbuernetes cluster",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdComplianceStart(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceStatus(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceResults(f, out, errOut))

	return cmd
}

// Run implements the compliance root command
func (o *ComplianceOptions) Run() error {
	return o.Cmd.Help()
}
