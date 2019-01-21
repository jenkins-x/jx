package cmd

import (
	"io"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
	commoncmd.CommonOptions
}

// NewCompliance creates a command object for the generic "compliance" action, which
// executes the compliance tests against a Kubernetes cluster
func NewCompliance(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ComplianceOptions{
		CommonOptions: commoncmd.CommonOptions{
			Factory: f,
			In:      in,
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
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdComplianceStatus(f, in, out, errOut))
	cmd.AddCommand(NewCmdComplianceResults(f, in, out, errOut))
	cmd.AddCommand(NewCmdComplianceRun(f, in, out, errOut))
	cmd.AddCommand(NewCmdComplianceDelete(f, in, out, errOut))
	cmd.AddCommand(NewCmdComplianceLogs(f, in, out, errOut))

	return cmd
}

// Run implements the compliance root command
func (o *ComplianceOptions) Run() error {
	return o.Cmd.Help()
}
