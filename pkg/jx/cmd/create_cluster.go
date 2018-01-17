package cmd

import (
	"io"
	"os/exec"

	"runtime"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"strings"
)

type KubernetesProvider string

// CreateClusterOptions the flags for running crest cluster
type CreateClusterOptions struct {
	CreateOptions
	Flags    InitFlags
	Provider string
}

const (
	GKE      string = "gke"
	EKS      string = "eks"
	AKS      string = "aks"
	MINIKUBE string = "minikube"
)

var KUBERNETES_PROVIDERS = map[string]bool{
	GKE:      true,
	EKS:      true,
	AKS:      true,
	MINIKUBE: true,
}

const (
	valid_providers = `Valid kubernetes providers include:

    * minikube (single-node Kubernetes cluster inside a VM on your laptop)
    * gke (Google Container Engine - https://cloud.google.com/kubernetes-engine)
    * aks (Azure Container Service - https://docs.microsoft.com/en-us/azure/aks)
    * coming soon:
        eks (Amazon Elastic Container Service - https://aws.amazon.com/eks)
    `
)

type CreateClusterFlags struct {
}

var (
	createClusterLong = templates.LongDesc(`
		This command creates a new kubernetes cluster, installing required local dependencies and provisions the Jenkins X platform

		%s

		Depending on which cloud provider your cluster is created on possible dependencies that will be installed are:

		- kubectl (CLI to interact with kubernetes clusters)
		- helm (package manager for kubernetes)
		- draft (CLI that makes it easy to build applications that run on kubernetes)\n
		- minikube (single-node Kubernetes cluster inside a VM on your laptop )\n
		- virtualisation drivers (to run minikube in a VM)\n
		- gcloud (Google Cloud CLI)
		- az (Azure CLI)

`)

	createClusterExample = templates.Examples(`

		jx create cluster minikube

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateCluster(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateClusterOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cluster [kubernetes provider]",
		Short:   "Create a new kubernetes cluster",
		Long:    fmt.Sprintf(createClusterLong, valid_providers),
		Example: createClusterExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateClusterMinikube(f, out, errOut))
	cmd.AddCommand(NewCmdCreateClusterGKE(f, out, errOut))
	cmd.AddCommand(NewCmdCreateClusterAKS(f, out, errOut))

	return cmd
}

func (o *CreateClusterOptions) Run() error {
	return o.Cmd.Help()
}

func (o *CreateClusterOptions) getClusterDependencies(deps []string) []string {

	d := getDependenciesToInstall("kubectl")
	if d != "" {
		deps = append(deps, d)
	}

	d = getDependenciesToInstall("helm")
	if d != "" {
		deps = append(deps, d)
	}

	d = getDependenciesToInstall("draft")
	if d != "" {
		deps = append(deps, d)
	}

	// Platform specific deps
	if runtime.GOOS == "darwin" {
		d = getDependenciesToInstall("brew")
		if d != "" {
			deps = append(deps, d)
		}
	}

	return deps
}
func (o *CreateClusterOptions) installMissingDependencies(providerSpecificDeps []string) error {
	// for now lets only support OSX until we can test on other platforms
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("jx create cluster currently only supported on OSX")
	}

	// get base list of required dependencies and add provider specific ones
	deps := o.getClusterDependencies(providerSpecificDeps)

	install := []string{}
	prompt := &survey.MultiSelect{
		Message: "Missing required dependencies, deselect to avoid auto installing:",
		Options: deps,
		Default: deps,
	}
	survey.AskOne(prompt, &install, nil)

	return o.doInstallMissingDependencies(install)
}

func (o *CreateClusterOptions) doInstallMissingDependencies(install []string) error {
	// install package managers first
	for _, i := range install {
		if i == "brew" {
			o.installBrew()
			break
		}
	}

	for _, i := range install {
		var err error
		switch i {
		case "kubectl":
			err = o.installKubectl()
		case "hyperkit":
			err = o.installHyperkit()
		case "xhyve":
			err = o.installXhyve()
		case "virtualbox":
			err = o.installVirtualBox()
		case "helm":
			err = o.installHelm()
		case "draft":
			err = o.installDraft()
		case "gcloud":
			err = o.installGcloud()
		case "minikube":
			err = o.installMinikube()
		case "az":
			err = o.installAzureCli()
		default:
			return fmt.Errorf("unknown dependency to install %s\n", i)
		}
		if err != nil {
			return fmt.Errorf("error installing %s\n", i)
		}
	}
	return nil
}

// appends the binary to the deps array if it cannot be found on the $PATH
func getDependenciesToInstall(d string) string {

	_, err := exec.LookPath(d)
	if err != nil {
		log.Infof("%s not found\n", d)
		return d
	}
	return ""
}

func (o *CreateClusterOptions) installBrew() error {
	return o.runCommand("/usr/bin/ruby", "-e", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)")
}

func (o *CreateClusterOptions) installKubectl() error {
	return o.runCommand("brew", "install", "kubectl")
}

func (o *CreateClusterOptions) installHyperkit() error {
	info, err := o.getCommandOutput("", "docker-machine-driver-hyperkit")
	if strings.Contains(info, "Docker") {
		o.Printf("docker-machine-driver-hyperkit is already installed\n")
		return nil
	}
	o.Printf("Result: %s and %v\n", info, err)
	err = o.runCommand("curl", "-LO", "https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-hyperkit")
	if err != nil {
		return err
	}

	err = o.runCommand("chmod", "+x", "docker-machine-driver-hyperkit")
	if err != nil {
		return err
	}

	log.Warn("Installing hyperkit does require sudo to perform some actions, for more details see https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver")

	err = o.runCommand("sudo", "mv", "docker-machine-driver-hyperkit", "/usr/local/bin/")
	if err != nil {
		return err
	}

	err = o.runCommand("sudo", "chown", "root:wheel", "/usr/local/bin/docker-machine-driver-hyperkit")
	if err != nil {
		return err
	}

	return o.runCommand("sudo", "chmod", "u+s", "/usr/local/bin/docker-machine-driver-hyperkit")
}

func (o *CreateClusterOptions) installVirtualBox() error {
	o.warnf("We cannot yet automate the installation of VirtualBox - can you install this manually please?\nPlease see: https://www.virtualbox.org/wiki/Downloads\n")
	return nil
}

func (o *CreateClusterOptions) installXhyve() error {
	info, err := o.getCommandOutput("", "brew", "info", "docker-machine-driver-xhyve")

	if err != nil || strings.Contains(info, "Not installed") {
		err = o.runCommand("brew", "install", "docker-machine-driver-xhyve")
		if err != nil {
			return err
		}

		brewPrefix, err := o.getCommandOutput("", "brew", "--prefix")
		if err != nil {
			return err
		}

		file := brewPrefix + "/opt/docker-machine-driver-xhyve/bin/docker-machine-driver-xhyve"
		err = o.runCommand("sudo", "chown", "root:wheel", file)
		if err != nil {
			return err
		}

		err = o.runCommand("sudo", "chmod", "u+s", file)
		if err != nil {
			return err
		}
		o.Printf("xhyve driver installed\n")
	} else {
		pgmPath, _ := exec.LookPath("docker-machine-driver-xhyve")
		o.Printf("xhyve driver is already available on your PATH at %s\n", pgmPath)
	}
	return nil
}

func (o *CreateClusterOptions) installHelm() error {
	return o.runCommand("brew", "install", "kubernetes-helm")
}

func (o *CreateClusterOptions) installDraft() error {
	err := o.runCommand("brew", "tap", "azure/draft")
	if err != nil {
		return err
	}

	return o.runCommand("brew", "install", "draft")
}

func (o *CreateClusterOptions) installMinikube() error {
	return o.runCommand("brew", "cask", "install", "minikube")
}

func (o *CreateClusterOptions) installGcloud() error {
	err := o.runCommand("brew", "tap", "caskroom/cask")
	if err != nil {
		return err
	}

	return o.runCommand("brew", "install", "google-cloud-sdk")
}

func (o *CreateClusterOptions) installAzureCli() error {
	return o.runCommand("brew", "install", "azure-cli")
}
