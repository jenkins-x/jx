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
type CreateClusterAKSOptions struct {
	CreateClusterOptions

	Flags CreateClusterAKSFlags
}

type CreateClusterAKSFlags struct {
}

var (
	createClusterAKSLong = templates.LongDesc(`
		This command creates a new kubernetes cluster on AKS, installing required local dependencies and provisions the
		Jenkins X platform

		Azure Container Service (AKS) manages your hosted Kubernetes environment, making it quick and easy to deploy and
		manage containerized applications without container orchestration expertise. It also eliminates the burden of
		ongoing operations and maintenance by provisioning, upgrading, and scaling resources on demand, without taking
		your applications offline.

`)

	createClusterAKSExample = templates.Examples(`

		jx create cluster aks

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateClusterAKS(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterAKSOptions{
		CreateClusterOptions: CreateClusterOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
			Provider: AKS,
		},
	}
	cmd := &cobra.Command{
		Use:     "aks",
		Short:   "Create a new kubernetes cluster on AKS: Runs on Azure",
		Long:    createClusterAKSLong,
		Example: createClusterAKSExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)

	//cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "4096", "Amount of RAM allocated to the minikube VM in MB")
	//cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "3", "Number of CPUs allocated to the minikube VM")

	return cmd
}

func (o *CreateClusterAKSOptions) Run() error {

	var deps []string
	d := binaryShouldBeInstalled("az")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("%v\nPlease fix the error or install manually then try again", err)
		os.Exit(-1)
	}

	err = o.createClusterAKS()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *CreateClusterAKSOptions) createClusterAKS() error {

	// TODO
	return fmt.Errorf("Create %s cluster not yet implemented", o.Provider)
}
