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

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type InitOptions struct {
	Factory  cmdutil.Factory
	Out      io.Writer
	Err      io.Writer
	Flags    InitFlags
	Provider KubernetesProvider
}

type KubernetesProvider string

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
		Factory: f,
		Out:     out,
		Err:     errOut,
	}

	cmd := &cobra.Command{
		Use:     "init [kubernetes provider]",
		Short:   "Init Jenkins-X",
		Long:    initLong,
		Example: initExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunInit(f, out, errOut, cmd, args, options)
			cmdutil.CheckErr(err)
		},
	}

	//cmd.Flags().StringP("git-provider", "", "GitHub", "Git provider, used to create tokens if not provided.  Supported providers: [GitHub]")

	// check if connected to a kube environment?

	return cmd
}

// RunInstall implements the generic Install command
func RunInit(f cmdutil.Factory, out, errOut io.Writer, cmd *cobra.Command, args []string, options *InitOptions) error {
	flags := InitFlags{
	//		Domain:             cmd.Flags().Lookup("domain").Value.String(),
	}
	options.Flags = flags

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

	e := exec.Command("/usr/bin/ruby", "-e", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installKubectl() error {
	e := exec.Command("brew", "install", "kubectl")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installHyperkit() error {

	e := exec.Command("curl", "-LO", "https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-hyperkit")
	e.Stdout = o.Out
	e.Stderr = o.Err

	e = exec.Command("chmod", "+x", "docker-machine-driver-hyperkit")
	e.Stdout = o.Out
	e.Stderr = o.Err

	log.Warn("Installing hyperkit does require sudo to perform some actions, fopr more details see https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver")

	e = exec.Command("sudo", "mv", "docker-machine-driver-hyperkit", "/usr/local/bin/")
	e.Stdout = o.Out
	e.Stderr = o.Err

	e = exec.Command("sudo", "chown", "root:wheel", "/usr/local/bin/docker-machine-driver-hyperkit")
	e.Stdout = o.Out
	e.Stderr = o.Err

	e = exec.Command("sudo", "chmod", "u+s", "/usr/local/bin/docker-machine-driver-hyperkit")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installHelm() error {
	e := exec.Command("brew", "install", "kubernetes-helm")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installDraft() error {
	e := exec.Command("brew", "tap", "azure/draft")
	e.Stdout = o.Out
	e.Stderr = o.Err
	e.Run()

	e = exec.Command("brew", "install", "draft")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installMinikube() error {
	e := exec.Command("brew", "cask", "install", "minikube")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *InitOptions) installGcloud() error {
	e := exec.Command("brew", "tap", "caskroom/cask")
	e.Stdout = o.Out
	e.Stderr = o.Err
	e.Run()

	e = exec.Command("brew", "install", "google-cloud-sdk")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}
func (o *InitOptions) installAzureCli() error {
	e := exec.Command("brew", "install", "azure-cli")
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}
