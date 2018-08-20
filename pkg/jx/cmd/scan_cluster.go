package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// ScanClusterOptions the options for 'scan cluster' command
type ScanClusterOptions struct {
	ScanOptions
}

// NewCmdScanCluster creates a command object for "scan cluster" command
func NewCmdScanCluster(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ScanClusterOptions{
		ScanOptions: ScanOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Performs a cluster security scan",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	return cmd
}

// Run executes the "scan cluster" command
func (o *ScanClusterOptions) Run() error {
	return o.Cmd.Help()
}
