package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

// Scan Options contains the command line options for scan commands
type ScanOptions struct {
	CommonOptions
}

var (
	scan_long = templates.LongDesc(`
		Perform a scan action.
	`)
)

// NewCmdScan creates a command object for the "scan" command
func NewCmdScan(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ScanOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Perform a scan action",
		Long:  scan_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdScanCluster(f, out, errOut))

	return cmd
}

func (o *ScanOptions) Run() error {
	return o.Cmd.Help()
}
