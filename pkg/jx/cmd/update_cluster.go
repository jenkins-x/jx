package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// UpdateClusterOptions the flags for running update cluster
type UpdateClusterOptions struct {
	UpdateOptions
	InstallOptions InstallOptions
	Flags          InitFlags
	Provider       string
}

type UpdateClusterFlags struct {
}

var (
	updateClusterLong = templates.LongDesc(`
		This command updates an existing Kubernetes cluster, it can be used to apply minor changes to a cluster / node pool

		%s

`)

	updateClusterExample = templates.Examples(`

		jx update cluster gke

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdUpdateCluster(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := createUpdateClusterOptions(f, in, out, errOut, "")

	cmd := &cobra.Command{
		Use:     "cluster [kubernetes provider]",
		Short:   "Updates an existing Kubernetes cluster",
		Long:    fmt.Sprintf(updateClusterLong, valid_providers),
		Example: updateClusterExample,
		Run: func(cmd2 *cobra.Command, args []string) {
			options.Cmd = cmd2
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateClusterGKE(f, in, out, errOut))

	return cmd
}

func createUpdateClusterOptions(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer, cloudProvider string) UpdateClusterOptions {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,

		Out: out,
		Err: errOut,
	}
	options := UpdateClusterOptions{
		UpdateOptions: UpdateOptions{
			CommonOptions: commonOptions,
		},
		Provider: cloudProvider,
	}
	return options
}

func (o *UpdateClusterOptions) Run() error {
	return o.Cmd.Help()
}
