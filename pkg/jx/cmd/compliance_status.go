package cmd

import (
	"fmt"
	"io"

	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	complianceStatusLong = templates.LongDesc(`
		Retrieves the current status of the compliance tests
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
	options := &ComplianceStatusOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Retrieves the status of compliance tests",
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
	cc, err := o.Factory.CreateComplianceClient()
	if err != nil {
		return errors.Wrap(err, "could not create the compliance client")
	}
	status, err := cc.GetStatus(complianceNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the status")
	}
	log.Info(hummanReadableStatus(status.Status))
	return nil
}

func hummanReadableStatus(status string) string {
	switch status {
	case aggregation.RunningStatus:
		return "Compliance tests are still running, it can take up to 60 minutes."
	case aggregation.FailedStatus:
		return "Compliance tests have failed. You can check what happened with `jx compliance results`."
	case aggregation.CompleteStatus:
		return "Compliance tests completed. Use `jx compliance results` to display the results."
	default:
		return fmt.Sprintf("Compliance tests are in unknown state %q.", status)
	}
}
