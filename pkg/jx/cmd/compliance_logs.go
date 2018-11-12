package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	bufSize = 2048
)

var (
	complianceLogsLongs = templates.LongDesc(`
		Prints the logs of compliance tests
	`)

	complianceLogsExample = templates.Examples(`
		# Print the compliance logs
		jx compliance logs
	`)
)

// ComplianceLogsOptions options for "compliance logs" command
type ComplianceLogsOptions struct {
	*CommonOptions

	Follow bool
}

// NewCmdComplianceLogs creates a command object for the "compliance logs" action, which
// prints the logs of compliance tests
func NewCmdComplianceLogs(commonOpts *CommonOptions) *cobra.Command {
	options := &ComplianceLogsOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "Prints the logs of compliance tests",
		Long:    complianceLogsLongs,
		Example: complianceLogsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Follow, "follow", "f", false, "Specify if the logs should be streamed.")

	return cmd
}

// Run implements the "compliance logs" command
func (o *ComplianceLogsOptions) Run() error {
	cc, err := o.Factory.CreateComplianceClient()
	if err != nil {
		return errors.Wrap(err, "could not create the compliance client")
	}
	logConfig := &client.LogConfig{
		Follow:    o.Follow,
		Namespace: complianceNamespace,
		Out:       os.Stdout,
	}
	logReader, err := cc.LogReader(logConfig)
	if err != nil {
		return errors.Wrap(err, "could not create the logs reader")
	}

	b := make([]byte, bufSize)
	for {
		n, err := logReader.Read(b)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "error reading the logs")
		}
		fmt.Fprint(logConfig.Out, string(b[:n]))
		if err == io.EOF {
			return nil
		}
	}
}
