package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"io"
)

// CreateClusterOptions the flags for running create cluster
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
		This command updates an existing kubernetes cluster, it can be used to apply minor changes to a cluster / node pool

		%s

`)

	updateClusterExample = templates.Examples(`

		jx update cluster gke

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdUpdateCluster(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := createUpdateClusterOptions(f, out, errOut, "")

	cmd := &cobra.Command{
		Use:     "cluster [kubernetes provider]",
		Short:   "Updates an existing kubernetes cluster",
		Long:    fmt.Sprintf(updateClusterLong, valid_providers),
		Example: updateClusterExample,
		Run: func(cmd2 *cobra.Command, args []string) {
			options.Cmd = cmd2
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateClusterGKE(f, out, errOut))

	return cmd
}

func createUpdateClusterOptions(f cmdutil.Factory, out io.Writer, errOut io.Writer, cloudProvider string) UpdateClusterOptions {
	commonOptions := CommonOptions{
		Factory: f,
		Out:     out,
		Err:     errOut,
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
