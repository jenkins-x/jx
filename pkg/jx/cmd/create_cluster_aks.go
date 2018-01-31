package cmd

import (
	"io"
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"strings"
	"github.com/Pallinder/go-randomdata"
)

// CreateClusterOptions the flags for running crest cluster
type CreateClusterAKSOptions struct {
	CreateClusterOptions

	Flags CreateClusterAKSFlags
}

type CreateClusterAKSFlags struct {
	ClusterName     string
	ResourceName    string
	Location        string
	NodeCount       string
	KubeVersion     string
	PathToPublicKey string
}

var (
	createClusterAKSLong = templates.LongDesc(`
		This command creates a new kubernetes cluster on AKS, installing required local dependencies and provisions the
		Jenkins X platform

		Azure Container Service (AKS) manages your hosted Kubernetes environment, making it quick and easy to deploy and
		manage containerized applications without container orchestration expertise. It also eliminates the burden of
		ongoing operations and maintenance by provisioning, upgrading, and scaling resources on demand, without taking
		your applications offline.

		Please use a location local to you: you can retrieve this from the Azure portal or by 
		running "az provider list" in your terminal.

		Important: You will need an account on azure, with a storage account (https://portal.azure.com/#create/Microsoft.StorageAccount-ARM)
        and network (https://portal.azure.com/#create/Microsoft.VirtualNetwork-ARM) - both linked to the resource group you use
		to create the cluster in.
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

	cmd.Flags().StringVarP(&options.Flags.ResourceName, "resource group name", "n", "", "Name of the resource group")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, "clusterName", "c", "", "Name of the cluster")
	cmd.Flags().StringVarP(&options.Flags.Location, "location", "l", "eastus", "location to run cluster in")
	cmd.Flags().StringVarP(&options.Flags.NodeCount, "nodes", "o", "1", "node count")
	cmd.Flags().StringVarP(&options.Flags.KubeVersion, "K8Version", "v", "1.8.2", "kubernetes version")
	cmd.Flags().StringVarP(&options.Flags.PathToPublicKey, "PathToPublicRSAKey", "k", "", "pathToPublicRSAKey")

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

	resourceName := o.Flags.ResourceName
	if resourceName == "" {
		resourceName = strings.ToLower(randomdata.SillyName())
		log.Infof("No resource name provided so using a generated one: %s", resourceName)
	}


	clusterName := o.Flags.ClusterName
	if clusterName == "" {
		clusterName = resourceName +"-cluster"
		log.Infof("No cluster name provided so using a generated one: %s", clusterName)
	}

	location := o.Flags.Location
	prompt := &survey.Input{
		Message: "location",
		Default: location,
		Help:    "location to run cluster",
	}
	survey.AskOne(prompt, &location, nil)


	nodeCount := o.Flags.NodeCount
	prompt = &survey.Input{
		Message: "nodes",
		Default: nodeCount,
		Help:    "number of nodes",
	}
	survey.AskOne(prompt, &nodeCount, nil)

	kubeVersion := o.Flags.KubeVersion
	prompt = &survey.Input{
		Message: "k8version",
		Default: kubeVersion,
		Help:    "k8 version",
	}
	survey.AskOne(prompt, &kubeVersion, nil)

	pathToPublicKey := o.Flags.PathToPublicKey

	//First login

	err := o.runCommand("az", "login")
	if err != nil {
		return err
	}

	//register for Microsoft Compute and Containers

	err = o.runCommand("az", "provider", "register", "-n", "Microsoft.Compute")
	if err != nil {
		return err
	}

	err = o.runCommand("az", "provider", "register", "-n", "Microsoft.ContainerService")
	if err != nil {
		return err
	}

	//create a resource group

	createGroup := []string{"group", "create", "-l", location, "-n", resourceName}

	err = o.runCommand("az", createGroup...)

	if err != nil {
		return err
	}

	createCluster := []string{"aks", "create", "-g", resourceName, "-n", clusterName, "-k", kubeVersion, "--node-count", nodeCount}

	if pathToPublicKey != "" {
		createCluster = append(createCluster, "--ssh-key-value", pathToPublicKey)
	} else {
		createCluster = append(createCluster, "--generate-ssh-keys")
	}

	err = o.runCommand("az", createCluster...)
	if err != nil {
		return err
	}

	//setup the kube context

	getCredentials := []string{"aks", "get-credentials", "--resource-group", resourceName, "--name", clusterName}

	err = o.runCommand("az", getCredentials...)

	if err != nil {
		return err
	}

	/**

	//create Container Registry on azure
	registryName := resourceName + "Registry"+randomdata.StringNumber(2,"")

	installContainerRegistry := []string{"acr","create","--resource-group",resourceName,"--name",registryName,"--sku Basic"}

	err = o.runCommand("az",installContainerRegistry...)

	if err != nil {
		return err
	}

	//login to the registry

	loginRegistry := []string{"acr","login","--name",registryName}

	err = o.runCommand("az",loginRegistry...)

	if err != nil {
		return err
	}

	**/


	// call jx init
	initOpts := &InitOptions{
		CommonOptions: o.CommonOptions,
		Provider:      AKS,
	}
	err = initOpts.Run()
	if err != nil {
		return err
	}

	// call jx install
	installOpts := &InstallOptions{
		CommonOptions:      o.CommonOptions,
		CloudEnvRepository: DEFAULT_CLOUD_ENVIRONMENTS_URL,
		Provider:           AKS, // TODO replace with context, maybe get from ~/.jx/gitAuth.yaml?
	}
	err = installOpts.Run()
	if err != nil {
		return err
	}

	return nil
}
