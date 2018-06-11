package cmd

import (
	"io"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// TODO change the name of the namespace to something more jx specific (e.g. jx-compliance), but
// at this time the Sonobuoy does not run properly into a custom namespace.
const complianceNamespace = "heptio-sonobuoy"

// kubeConformanceImage is the URL of the docker image to run for the kube conformance tests
const kubeConformanceImage = "gcr.io/heptio-images/kube-conformance:latest"

// compliance is the URL of the docker image to run for the Sonobuoy aggregator and workers
var complianceImage = "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version

// ComplianceOptions options for compliance command
type ComplianceOptions struct {
	CommonOptions
}

// NewCompliance creates a command object for the generic "compliance" action, which
// executes the compliance tests against a Kubernetes cluster
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
		Short: "Run compliance tests against Kubernetes cluster",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdComplianceStatus(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceResults(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceRun(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceDelete(f, out, errOut))
	cmd.AddCommand(NewCmdComplianceLogs(f, out, errOut))

	return cmd
}

// Run implements the compliance root command
func (o *ComplianceOptions) Run() error {
	return o.Cmd.Help()
}
