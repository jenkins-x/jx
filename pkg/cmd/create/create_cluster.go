package create

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/cmd/create/options"

	"github.com/jenkins-x/jx/pkg/cmd/initcmd"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
)

type KubernetesProvider string

// CreateClusterOptions the flags for running create cluster
type CreateClusterOptions struct {
	options.CreateOptions
	InstallOptions InstallOptions
	Flags          initcmd.InitFlags
	Provider       string
}

const (
	optionKubernetesVersion = "kubernetes-version"
	optionNodes             = "nodes"
	optionCluster           = "cluster"
	optionClusterName       = "cluster-name"
	optionCloudProvider     = "cloud-provider"
)

const (
	ValidKubernetesProviders = `Valid Kubernetes providers include:

    * aks (Azure Container Service - https://docs.microsoft.com/en-us/azure/aks)
    * aws (Amazon Web Services via kops - https://github.com/aws-samples/aws-workshop-for-kubernetes/blob/master/readme.adoc)
    * eks (Amazon Web Services Elastic Container Service for Kubernetes - https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)
    * gke (Google Container Engine - https://cloud.google.com/kubernetes-engine)
    # icp (IBM Cloud Private) - https://www.ibm.com/cloud/private
    * iks (IBM Cloud Kubernetes Service - https://console.bluemix.net/docs/containers)
    * oke (Oracle Cloud Infrastructure Container Engine for Kubernetes - https://docs.cloud.oracle.com/iaas/Content/ContEng/Concepts/contengoverview.htm)
    * kubernetes for custom installations of Kubernetes
    * minikube (single-node Kubernetes cluster inside a VM on your laptop)
	* minishift (single-node OpenShift cluster inside a VM on your laptop)
	* openshift for installing on 3.9.x or later clusters of OpenShift
`
)

type CreateClusterFlags struct {
}

var (
	CreateClusterLong = templates.LongDesc(`
		This command creates a new Kubernetes cluster, installing required local dependencies and provisions the Jenkins X platform

		You can see a demo of this command here: [https://jenkins-x.io/demos/create_cluster/](https://jenkins-x.io/demos/create_cluster/)

		%s

		Depending on which cloud provider your cluster is created on possible dependencies that will be installed are:

		- kubectl (CLI to interact with Kubernetes clusters)
		- helm (package manager for Kubernetes)
		- draft (CLI that makes it easy to build applications that run on Kubernetes)
		- minikube (single-node Kubernetes cluster inside a VM on your laptop )
		- minishift (single-node OpenShift cluster inside a VM on your laptop)
		- virtualisation drivers (to run Minikube in a VM)
		- gcloud (Google Cloud CLI)
		- oci (Oracle Cloud Infrastructure CLI)
		- az (Azure CLI)
		- ibmcloud (IBM CLoud CLI)

		For more documentation see: [https://jenkins-x.io/getting-started/create-cluster/](https://jenkins-x.io/getting-started/create-cluster/)

`)

	CreateClusterExample = templates.Examples(`

		# create a cluster on Google Cloud
		jx create cluster gke --skip-installation

		# create a cluster on AWS via EKS
		jx create cluster eks --skip-installation
`)
)

// NewCmdCreateCluster creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdCreateCluster(commonOpts *opts.CommonOptions) *cobra.Command {
	options := createCreateClusterOptions(commonOpts, "")

	cmd := &cobra.Command{
		Use:     "cluster [kubernetes provider]",
		Short:   "Create a new Kubernetes cluster",
		Long:    fmt.Sprintf(CreateClusterLong, ValidKubernetesProviders),
		Example: CreateClusterExample,
		Run: func(cmd2 *cobra.Command, args []string) {
			options.Cmd = cmd2
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateClusterAKS(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterAWS(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterEKS(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterGKE(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterMinikube(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterMinishift(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterOKE(commonOpts))
	cmd.AddCommand(NewCmdCreateClusterIKS(commonOpts))

	return cmd
}

func (o *CreateClusterOptions) addCreateClusterFlags(cmd *cobra.Command) {
	o.InstallOptions.AddInstallFlags(cmd, true)
}

func createCreateClusterOptions(commonOpts *opts.CommonOptions, cloudProvider string) CreateClusterOptions {
	options := CreateClusterOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: commonOpts,
		},
		Provider:       cloudProvider,
		InstallOptions: CreateInstallOptions(commonOpts),
	}
	return options
}

// Run returns help if function is run without any argument
func (o *CreateClusterOptions) Run() error {
	return o.Cmd.Help()
}
