package cmd

import (
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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

func NewCmdUpdateClusterGKE(commonOpts *CommonOptions) *cobra.Command {
	options := createUpdateClusterGKEOptions(commonOpts, cloud.GKE)

	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Updates an existing Kubernetes cluster on GKE: Runs on Google Cloud",
		Long:    updateClusterGKELong,
		Example: updateClusterGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateClusterGKETerraform(commonOpts))

	return cmd
}

func createUpdateClusterGKEOptions(commonOpts *CommonOptions, cloudProvider string) UpdateClusterGKEOptions {
	options := UpdateClusterGKEOptions{
		UpdateClusterOptions: UpdateClusterOptions{
			UpdateOptions: UpdateOptions{
				CommonOptions: commonOpts,
			},
			Provider: cloudProvider,
		},
	}
	return options
}

func (o *UpdateClusterGKEOptions) Run() error {
	return o.Cmd.Help()
}
