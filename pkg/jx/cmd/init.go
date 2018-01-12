package cmd

import (
	"io"
	"os"
	"os/exec"

	"runtime"

	"fmt"

	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

type KubernetesProvider string

// InitOptions the flags for running init
type InitOptions struct {
	CommonOptions

	Flags    InitFlags
	Provider KubernetesProvider
}

const (
	GKE      KubernetesProvider = "gke"
	EKS      KubernetesProvider = "eks"
	AKS      KubernetesProvider = "aks"
	MINIKUBE KubernetesProvider = "minikube"
)
const (
	valid_providers = `Valid kubernetes providers include:

    * minikube (single-node Kubernetes cluster inside a VM on your laptop)
    * GKE (Google Container Engine - https://cloud.google.com/kubernetes-engine)
    * AKS (Azure Container Service - https://docs.microsoft.com/en-us/azure/aks)
    * coming soon:
        EKS (Amazon Elastic Container Service - https://aws.amazon.com/eks)
    `
)

type InitFlags struct {
}

var (
	initLong = templates.LongDesc(`
		This command installs the dependencies to run the Jenkins-X platform

		Dependencies include:
			- kubectl (CLI to interact with kubernetes clusters)
			- helm (package manager for kubernetes)
			- draft (CLI that makes it easy to build applications that run on kubernetes)
			- minikube (single-node Kubernetes cluster inside a VM on your laptop )
			- virtualisation drivers (to run minikube in a VM)

`)

	initExample = templates.Examples(`

		jx init minikube

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdInit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &InitOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "init [kubernetes provider]",
		Short:   "Init Jenkins-X",
		Long:    initLong,
		Example: initExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	//cmd.Flags().StringP("git-provider", "", "GitHub", "Git provider, used to create tokens if not provided.  Supported providers: [GitHub]")

	// check if connected to a kube environment?

	return cmd
}

func (options *InitOptions) Run() error {
	args := options.Args
	cmd := options.Cmd
	if len(args) != 1 {
		log.Errorf("You must specify one kubernetes provider you want to setup and configure, if unknown minikube is a great way to trial Jenkins-X locally. %s", valid_providers)

		usageString := "Required kubernetes provider."
		return cmdutil.UsageError(cmd, usageString)
	}
	kind := strings.ToLower(args[0])
	switch kind {
	case "gke":
		options.Provider = GKE

	case "aks":
		options.Provider = AKS
		//log.Errorf("AKS not yet supported, currently only GKE and minikube are available")
		//os.Exit(-1)

	case "eks":
		options.Provider = EKS
		log.Errorf("AKS not yet supported, currently only GKE and minikube are available")
		os.Exit(-1)

	case "minikube":
		options.Provider = MINIKUBE

	default:
		return cmdutil.UsageError(cmd, "Unknown kubernetes provider: %s", kind)
	}

	err := options.installMissingDependencies()
	if err != nil {
		log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}
	return nil
}

func (o *InitOptions) installMissingDependencies() error {
	// for now lets only support OSX until we can test on other platforms
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("jx init currently only supported on OSX")
	}

	var deps []string
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
		d = getDependenciesToInstall("hyperkit")
		if d != "" {
			deps = append(deps, d)
		}
	}

	if o.Provider == MINIKUBE {
		d = getDependenciesToInstall("minikube")
		if d != "" {
			deps = append(deps, d)
		}
	}

	if o.Provider == GKE {
		d = getDependenciesToInstall("gcloud")
		if d != "" {
			deps = append(deps, d)
		}
	}

	if o.Provider == AKS {
		d = getDependenciesToInstall("az")
		if d != "" {
			deps = append(deps, d)
		}
	}

	//TODO add eks cli

	install := []string{}
	prompt := &survey.MultiSelect{
		Message: "Missing dependencies, deselect to avoid auto installing:",
		Options: deps,
		Default: deps,
	}
	survey.AskOne(prompt, &install, nil)

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
			return fmt.Errorf("unknown dependency to install %s", i)
		}
		if err != nil {
			return fmt.Errorf("error installing %s", i)
		}
	}
	return nil
}

// appends the binary to the deps array if it cannot be found on the $PATH
func getDependenciesToInstall(d string) string {

	_, err := exec.LookPath(d)
	if err != nil {
		log.Infof("%s not found", d)
		return d
	}
	return ""
}

func (o *InitOptions) installBrew() error {
	return o.runCommand("/usr/bin/ruby", "-e", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)")
}

func (o *InitOptions) installKubectl() error {
	return o.runCommand("brew", "install", "kubectl")
}

func (o *InitOptions) installHyperkit() error {
	err := o.runCommand("curl", "-LO", "https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-hyperkit")
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

func (o *InitOptions) installHelm() error {
	return o.runCommand("brew", "install", "kubernetes-helm")
}

func (o *InitOptions) installDraft() error {
	err := o.runCommand("brew", "tap", "azure/draft")
	if err != nil {
		return err
	}

	return o.runCommand("brew", "install", "draft")
}

func (o *InitOptions) installMinikube() error {
	return o.runCommand("brew", "cask", "install", "minikube")
}

func (o *InitOptions) installGcloud() error {
	err := o.runCommand("brew", "tap", "caskroom/cask")
	if err != nil {
		return err
	}

	return o.runCommand("brew", "install", "google-cloud-sdk")
}

func (o *InitOptions) installAzureCli() error {
	return o.runCommand("brew", "install", "azure-cli")
}
