package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"time"

	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// CreateClusterOptions the flags for running create cluster
type CreateClusterMinikubeOptions struct {
	CreateClusterOptions

	Flags    CreateClusterMinikubeFlags
	Provider KubernetesProvider
}

type CreateClusterMinikubeFlags struct {
	Memory              string
	CPU                 string
	Driver              string
	HyperVVirtualSwitch string
	Namespace           string
	ClusterVersion      string
}

var (
	createClusterMinikubeLong = templates.LongDesc(`
		This command creates a new kubernetes cluster, installing required local dependencies and provisions the
		Jenkins X platform

		Minikube is a tool that makes it easy to run Kubernetes locally. Minikube runs a single-node Kubernetes
		cluster inside a VM on your laptop for users looking to try out Kubernetes or develop with it day-to-day.

`)

	createClusterMinikubeExample = templates.Examples(`

		jx create cluster minikube

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateClusterMinikube(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterMinikubeOptions{
		CreateClusterOptions: createCreateClusterOptions(f, out, errOut, MINIKUBE),
	}
	cmd := &cobra.Command{
		Use:     "minikube",
		Short:   "Create a new kubernetes cluster with minikube: Runs locally",
		Long:    createClusterMinikubeLong,
		Example: createClusterMinikubeExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "", fmt.Sprintf("Amount of RAM allocated to the minikube VM in MB. Defaults to %s MB.", MinikubeDefaultMemory))
	cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "", fmt.Sprintf("Number of CPUs allocated to the minikube VM. Defaults to %s.", MinikubeDefaultCpu))
	cmd.Flags().StringVarP(&options.Flags.Driver, "vm-driver", "d", "", "VM driver is one of: [hyperkit hyperv kvm kvm2 virtualbox vmwarefusion xhyve]")
	cmd.Flags().StringVarP(&options.Flags.HyperVVirtualSwitch, "hyperv-virtual-switch", "v", "", "Additional options for using HyperV with minikube")
	cmd.Flags().StringVarP(&options.Flags.ClusterVersion, optionKubernetesVersion, "", "", "kubernetes version")

	return cmd
}

func (o *CreateClusterMinikubeOptions) Run() error {
	var deps []string
	d := binaryShouldBeInstalled("minikube")
	if d != "" {
		deps = append(deps, d)
	}

	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	if o.isExistingMinikubeRunning() {
		log.Error("an existing minikube cluster is already running, perhaps use `jx install`.\nNote existing minikube must have RBAC enabled, running `minikube delete` and `jx create cluster minikube` creates a new VM with RBAC enabled")
		os.Exit(-1)
	}

	err = o.createClusterMinikube()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *CreateClusterMinikubeOptions) defaultMacVMDriver() string {
	_, err := o.getCommandOutput("", "hyperkit", "-v")
	if err != nil {
		o.warnf("Could not find hyperkit on your PATH. If you install Docker for Mac then we could use hyperkit.\nSee: https://docs.docker.com/docker-for-mac/install/\n")
		return "xhyve"
	}
	return "hyperkit"
}

func (o *CreateClusterMinikubeOptions) isExistingMinikubeRunning() bool {

	var cmd_out bytes.Buffer

	e := exec.Command("minikube", "status")
	e.Stdout = &cmd_out
	e.Stderr = o.Err
	err := e.Run()

	if err != nil {
		return false
	}

	if strings.Contains(cmd_out.String(), "Running") {
		return true
	}

	return false
}

func (o *CreateClusterMinikubeOptions) createClusterMinikube() error {

	mem := o.Flags.Memory
	prompt := &survey.Input{
		Message: "memory (MB)",
		Default: MinikubeDefaultMemory,
		Help:    "Amount of RAM allocated to the minikube VM in MB",
	}
	showPromptIfOptionNotSet(&mem, prompt)

	cpu := o.Flags.CPU
	prompt = &survey.Input{
		Message: "cpu (cores)",
		Default: MinikubeDefaultCpu,
		Help:    "Number of CPUs allocated to the minikube VM",
	}
	showPromptIfOptionNotSet(&cpu, prompt)

	vmDriverValue := o.Flags.Driver

	defaultDriver := ""
	if len(vmDriverValue) == 0 {
		switch runtime.GOOS {
		case "darwin":
			defaultDriver = o.defaultMacVMDriver()
		case "windows":
			defaultDriver = "hyperv"
		case "linux":
			defaultDriver = "kvm"
		default:
			defaultDriver = "virtualbox"
		}
	}

	// only add drivers that are appropriate for this OS
	drivers := []string{defaultDriver}
	if defaultDriver == "kvm" {
		drivers = append(drivers, "kvm2")
	}
	if defaultDriver != "virtualbox" {
		drivers = append(drivers, "virtualbox")
	}
	if runtime.GOOS == "darwin" {
		if util.StringArrayIndex(drivers, "xhyve") < 0 {
			drivers = append(drivers, "xhyve")
		}
	}
	if runtime.GOOS == "linux" {
		drivers = append(drivers, "none")
	}

	prompts := &survey.Select{
		Message: "Select driver:",
		Options: drivers,
		Default: defaultDriver,
		Help:    "VM driver, defaults to recommended native virtualisation",
	}

	showPromptIfOptionNotSet(&vmDriverValue, prompts)

	if vmDriverValue != "none" {
		err := o.doInstallMissingDependencies([]string{vmDriverValue})
		if err != nil {
			log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
			os.Exit(-1)
		}
	}

	args := []string{"start", "--memory", mem, "--cpus", cpu, "--vm-driver", vmDriverValue, "--bootstrapper=kubeadm"}
	hyperVVirtualSwitch := o.Flags.HyperVVirtualSwitch
	if hyperVVirtualSwitch != "" {
		args = append(args, "--hyperv-virtual-switch", hyperVVirtualSwitch)
	}
	kubernetesVersion := o.Flags.ClusterVersion
	if kubernetesVersion != "" {
		args = append(args, "--kubernetes-version", kubernetesVersion)
	}
	o.Out.Write([]byte("Creating Minikube cluster...\n"))
	err := o.runCommand("minikube", args...)
	if err != nil {
		return err
	} else {
		o.Out.Write([]byte("Minikube cluster created.\n"))
	}

	err = o.retry(3, 10*time.Second, func() (err error) {
		err = o.runCommand("kubectl", "create", "clusterrolebinding", "add-on-cluster-admin", "--clusterrole", "cluster-admin", "--serviceaccount", "kube-system:default")
		return
	})
	if err != nil {
		return err
	}

	ip, err := o.getCommandOutput("", "minikube", "ip")
	if err != nil {
		return err
	}
	o.InstallOptions.Flags.Domain = ip + ".nip.io"

	log.Info("Initialising cluster ...\n")
	err = o.initAndInstall(MINIKUBE)
	if err != nil {
		return err
	}

	context, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	ns := o.Flags.Namespace
	if ns == "" {
		f := o.Factory
		_, ns, _ = f.CreateClient()
		if err != nil {
			return err
		}
	}

	err = o.runCommand("kubectl", "config", "set-context", context, "--namespace", ns)
	if err != nil {
		return err
	}

	err = o.runCommand("kubectl", "get", "ingress")
	if err != nil {
		return err
	}

	return nil
}

func showPromptIfOptionNotSet(option *string, p survey.Prompt) error {
	if *option == "" {
		err := survey.AskOne(p, option, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
