package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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

func NewCmdUpdateClusterGKE(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := createUpdateClusterGKEOptions(f, in, out, errOut, commoncmd.GKE)

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

	cmd.AddCommand(NewCmdUpdateClusterGKETerraform(f, in, out, errOut))

	return cmd
}

func createUpdateClusterGKEOptions(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer, cloudProvider string) UpdateClusterGKEOptions {
	commonOptions := commoncmd.CommonOptions{
		Factory: f,
		In:      in,
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
