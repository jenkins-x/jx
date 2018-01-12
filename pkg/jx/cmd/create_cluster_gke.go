package cmd

import (
	"io"
	"os"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateClusterOptions the flags for running crest cluster
type CreateClusterGKEOptions struct {
	CreateClusterOptions

	Flags CreateClusterGKEFlags
}

type CreateClusterGKEFlags struct {
}

var (
	createClusterGKELong = templates.LongDesc(`
		This command creates a new kubernetes cluster on GKE, installing required local dependencies and provisions the
		Jenkins-X platform

		Google Kubernetes Engine is a managed environment for deploying containerized applications. It brings our latest
		innovations in developer productivity, resource efficiency, automated operations, and open source flexibility to
		accelerate your time to market.

		Google has been running production workloads in containers for over 15 years, and we build the best of what we
		learn into Kubernetes, the industry-leading open source container orchestrator which powers Kubernetes Engine.

`)

	createClusterGKEExample = templates.Examples(`

		jx create cluster gke

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateClusterGKE(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterGKEOptions{
		CreateClusterOptions: CreateClusterOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
			Provider: GKE,
		},
	}
	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Create a new kubernetes cluster on GKE",
		Long:    createClusterGKELong,
		Example: createClusterGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	//cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "4096", "Amount of RAM allocated to the minikube VM in MB")
	//cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "3", "Number of CPUs allocated to the minikube VM")

	return cmd
}

func (o *CreateClusterGKEOptions) Run() error {

	var deps []string
	d := getDependenciesToInstall("gcloud")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	err = o.createClusterGKE()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *CreateClusterGKEOptions) createClusterGKE() error {

	// TODO
	return fmt.Errorf("Create %s cluster not yet implemented", o.Provider)
}
