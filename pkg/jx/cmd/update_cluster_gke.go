package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateClusterOptions the flags for running create cluster
type UpdateClusterGKEOptions struct {
	UpdateClusterOptions
}

var (
	updateClusterGKELong = templates.LongDesc(`
		
		Not currently implemented.

`)

	updateClusterGKEExample = templates.Examples(`

		jx update cluster gke

`)
)

func NewCmdUpdateClusterGKE(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := createUpdateClusterGKEOptions(f, out, errOut, GKE)

	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Updates an existing kubernetes cluster on GKE: Runs on Google Cloud",
		Long:    updateClusterGKELong,
		Example: updateClusterGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateClusterGKETerraform(f, out, errOut))

	return cmd
}

func createUpdateClusterGKEOptions(f cmdutil.Factory, out io.Writer, errOut io.Writer, cloudProvider string) UpdateClusterGKEOptions {
	commonOptions := CommonOptions{
		Factory: f,
		Out:     out,
		Err:     errOut,
	}
	options := UpdateClusterGKEOptions{
		UpdateClusterOptions: UpdateClusterOptions{
			UpdateOptions: UpdateOptions{
				CommonOptions: commonOptions,
			},
			Provider: cloudProvider,
		},
	}
	return options
}

func (o *UpdateClusterGKEOptions) Run() error {
	return o.Cmd.Help()
}
