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
	"path/filepath"
)

// CreateClusterOptions the flags for running crest cluster
type CreateClusterAKSOptions struct {
	CreateClusterOptions

	Flags CreateClusterAKSFlags
}

type CreateClusterAKSFlags struct {
	UserName        string
	Password        string
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
	cmd.Flags().StringVarP(&options.Flags.NodeCount, "nodes", "o", "", "node count")
	cmd.Flags().StringVarP(&options.Flags.KubeVersion, optionKubernetesVersion, "v", "1.9.1", "kubernetes version")
	cmd.Flags().StringVarP(&options.Flags.PathToPublicKey, "path-To-public-rsa-key", "k", "", "pathToPublicRSAKey")
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
		clusterName = resourceName + "-cluster"
		log.Infof("No cluster name provided so using a generated one: %s", clusterName)
	}

	location := o.Flags.Location
	if location == "" {
		prompt := &survey.Select{
			Message:  "location",
			Options:  aks.GetResourceGrouoLocation(),
			Default:  "eastus",
			PageSize: 10,
			Help:     "location to run cluster",
		}
		err := survey.AskOne(prompt, &location, nil)
		if err != nil {
			return err
		}
	}

	nodeCount := o.Flags.NodeCount
	if nodeCount == "" {
		prompt := &survey.Input{
			Message: "nodes",
			Default: "3",
			Help:    "number of nodes",
		}
		survey.AskOne(prompt, &nodeCount, nil)
	}

	kubeVersion := o.Flags.KubeVersion
	if kubeVersion == "" {
		prompt := &survey.Input{
			Message: "k8version",
			Default: kubeVersion,
			Help:    "k8 version",
		}
		survey.AskOne(prompt, &kubeVersion, nil)
	}

	pathToPublicKey := o.Flags.PathToPublicKey

	userName := o.Flags.UserName
	password := o.Flags.Password

	//First login

	var err error
	if userName != "" && password != "" {
		err = o.runCommand("az", "login", "-u", userName, "-p", password)
		if err != nil {
			return err
		}
	} else {
		err = o.runCommand("az", "login")
		if err != nil {
			return err
		}
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
	 * create a cluster admin role
	 */

	err = o.createClusterAdmin()
	if err != nil {
		msg :=err.Error()
		if strings.Contains(msg,"AlreadyExists"){
			log.Success("role cluster-admin already exists for the cluster")

		}else {
			return err
		}
	}else{
		log.Success("created role cluster-admin")
	}


	return o.initAndInstall(AKS)
}

func (o *CreateClusterAKSOptions)createClusterAdmin() error {

	content := []byte(
		`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: cluster-admin
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - '*'
- nonResourceURLs:
  - '*'
  verbs:
  - '*'`)

	fileName := randomdata.SillyName() + ".yml"
	fileName = filepath.Join(os.TempDir(),fileName)
	tmpfile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	err = o.runCommand("kubectl", "create", "clusterrolebinding", "kube-system-cluster-admin", "--clusterrole", "cluster-admin", "--serviceaccount", "kube-system:default")
	if err != nil{
		return err
	}
	err = o.runCommand("kubectl", "create","-f", tmpfile.Name())
	if err != nil{
		return err
	}
	return nil
}
