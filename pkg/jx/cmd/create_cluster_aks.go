package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/jx/cmd/aks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// CreateClusterOptions the flags for running create cluster
type CreateClusterAKSOptions struct {
	CreateClusterOptions

	Flags CreateClusterAKSFlags
}

type CreateClusterAKSFlags struct {
	UserName                  string
	Password                  string
	ClusterName               string
	ResourceName              string
	Location                  string
	NodeVMSize                string
	NodeOSDiskSize            string
	NodeCount                 string
	KubeVersion               string
	PathToPublicKey           string
	SkipLogin                 bool
	SkipProviderRegistration  bool
	SkipResourceGroupCreation bool
	Tags                      string
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
		CreateClusterOptions: createCreateClusterOptions(f, out, errOut, AKS),
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

	cmd.Flags().StringVarP(&options.Flags.UserName, "user name", "u", "", "user name")
	cmd.Flags().StringVarP(&options.Flags.Password, "password", "p", "", "password")
	cmd.Flags().StringVarP(&options.Flags.ResourceName, "resource-group-name", "n", "", "Name of the resource group")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, "cluster-name", "c", "", "Name of the cluster")
	cmd.Flags().StringVarP(&options.Flags.Location, "location", "l", "", "location to run cluster in")
	cmd.Flags().StringVarP(&options.Flags.NodeVMSize, "node-vm-size", "s", "", "Size of Virtual Machines to create as Kubernetes nodes")
	cmd.Flags().StringVarP(&options.Flags.NodeOSDiskSize, "disk-size", "", "", "Size in GB of the OS disk for each node in the node pool.")
	cmd.Flags().StringVarP(&options.Flags.NodeCount, "nodes", "o", "", "node count")
	cmd.Flags().StringVarP(&options.Flags.KubeVersion, optionKubernetesVersion, "v", "", "Version of Kubernetes to use for creating the cluster, such as '1.8.11' or '1.9.6'.  Values from: `az aks get-versions`.")
	cmd.Flags().StringVarP(&options.Flags.PathToPublicKey, "path-To-public-rsa-key", "k", "", "pathToPublicRSAKey")
	cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip login if already logged in using `az login`")
	cmd.Flags().BoolVarP(&options.Flags.SkipProviderRegistration, "skip-provider-registration", "", false, "Skip provider registration")
	cmd.Flags().BoolVarP(&options.Flags.SkipResourceGroupCreation, "skip-resource-group-creation", "", false, "Skip resource group creation")
	cmd.Flags().StringVarP(&options.Flags.Tags, "tags", "", "", "Space-separated tags in 'key[=value]' format. Use '' to clear existing tags.")
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
		clusterName = strings.ToLower(randomdata.SillyName())
		log.Infof("No cluster name provided so using a generated one: %s", clusterName)
	}

	location := o.Flags.Location
	if location == "" {
		prompt := &survey.Select{
			Message:  "Location",
			Options:  aks.GetResourceGroupLocation(),
			Default:  "eastus",
			PageSize: 10,
			Help:     "location to run cluster",
		}
		err := survey.AskOne(prompt, &location, nil)
		if err != nil {
			return err
		}
	}

	nodeVMSize := o.Flags.NodeVMSize
	if nodeVMSize == "" {
		prompts := &survey.Select{
			Message:  "Virtual Machine Size:",
			Options:  aks.GetSizes(),
			Help:     "We recommend a minimum of Standard_D2s_v3 for Jenkins X.\nA table of machine descriptions can be found here https://azure.microsoft.com/en-us/pricing/details/virtual-machines/linux/",
			PageSize: 10,
			Default:  "Standard_D2s_v3",
		}

		err := survey.AskOne(prompts, &nodeVMSize, nil)
		if err != nil {
			return err
		}
	}

	nodeCount := o.Flags.NodeCount
	if nodeCount == "" {
		prompt := &survey.Input{
			Message: "Number of Nodes",
			Default: "3",
			Help:    "We recommend a minimum of 3 nodes for Jenkins X",
		}
		survey.AskOne(prompt, &nodeCount, nil)
	}

	pathToPublicKey := o.Flags.PathToPublicKey

	userName := o.Flags.UserName
	password := o.Flags.Password

	var err error
	if !o.Flags.SkipLogin {
		//First login

		if userName != "" && password != "" {
			log.Info("Logging in to Azure using provider username and password...\n")
			err = o.runCommand("az", "login", "-u", userName, "-p", password)
			if err != nil {
				return err
			}
		} else {
			log.Info("Logging in to Azure interactively...\n")
			err = o.runCommandVerbose("az", "login")
			if err != nil {
				return err
			}
		}
	}

	if !o.Flags.SkipProviderRegistration {
		//register for Microsoft Compute and Containers

		err = o.runCommand("az", "provider", "register", "-n", "Microsoft.Compute")
		if err != nil {
			return err
		}

		err = o.runCommand("az", "provider", "register", "-n", "Microsoft.ContainerService")
		if err != nil {
			return err
		}
	}

	if !o.Flags.SkipResourceGroupCreation {
		//create a resource group

		createGroup := []string{"group", "create", "-l", location, "-n", resourceName}

		err = o.runCommand("az", createGroup...)

		if err != nil {
			return err
		}
	}

	createCluster := []string{"aks", "create", "-g", resourceName, "-n", clusterName}

	if o.Flags.KubeVersion != "" {
		createCluster = append(createCluster, "--kubernetes-version", o.Flags.KubeVersion)
	}

	if nodeVMSize != "" {
		createCluster = append(createCluster, "--node-vm-size", nodeVMSize)
	}

	if o.Flags.NodeOSDiskSize != "" {
		createCluster = append(createCluster, "--node-osdisk-size", o.Flags.NodeOSDiskSize)
	}

	if nodeCount != "" {
		createCluster = append(createCluster, "--node-count", nodeCount)
	}

	if pathToPublicKey != "" {
		createCluster = append(createCluster, "--ssh-key-value", pathToPublicKey)
	} else {
		createCluster = append(createCluster, "--generate-ssh-keys")
	}

	if o.Flags.Tags != "" {
		createCluster = append(createCluster, "--tags", o.Flags.Tags)
	}

	log.Infof("Creating cluster named %s in resource group %s...\n", clusterName, resourceName)
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

	log.Info("Initialising cluster ...\n")
	return o.initAndInstall(AKS)
}
