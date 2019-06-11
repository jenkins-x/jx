package create

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"time"

	"fmt"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/features"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
	DiskSize            string
	Driver              string
	HyperVVirtualSwitch string
	Namespace           string
	ClusterVersion      string
}

const (
	MinikubeDefaultCpu = "3"

	MinikubeDefaultDiskSize = "150GB"

	MinikubeDefaultMemory = "4096"
)

var (
	createClusterMinikubeLong = templates.LongDesc(`
		This command creates a new Kubernetes cluster, installing required local dependencies and provisions the
		Jenkins X platform

		Minikube is a tool that makes it easy to run Kubernetes locally. Minikube runs a single-node Kubernetes
		cluster inside a VM on your laptop for users looking to try out Kubernetes or develop with it day-to-day.

`)

	createClusterMinikubeExample = templates.Examples(`

		jx create cluster minikube

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdCreateClusterMinikube(commonOpts *opts.CommonOptions) *cobra.Command {
	options := CreateClusterMinikubeOptions{
		CreateClusterOptions: createCreateClusterOptions(commonOpts, cloud.MINIKUBE),
	}
	cmd := &cobra.Command{
		Use:     "minikube",
		Short:   "Create a new Kubernetes cluster with Minikube: Runs locally",
		Long:    createClusterMinikubeLong,
		Example: createClusterMinikubeExample,
		PreRun: func(cmd *cobra.Command, args []string) {
			err := features.IsEnabled(cmd)
			helper.CheckErr(err)
			err = options.InstallOptions.CheckFeatures()
			helper.CheckErr(err)
		},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "", fmt.Sprintf("Amount of RAM allocated to the Minikube VM in MB. Defaults to %s MB.", MinikubeDefaultMemory))
	cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "", fmt.Sprintf("Number of CPUs allocated to the Minikube VM. Defaults to %s.", MinikubeDefaultCpu))
	cmd.Flags().StringVarP(&options.Flags.DiskSize, "disk-size", "s", "", fmt.Sprintf("Total amount of storage allocated to the Minikube VM. Defaults to %s", MinikubeDefaultDiskSize))
	cmd.Flags().StringVarP(&options.Flags.Driver, "vm-driver", "d", "", "VM driver is one of: [hyperkit hyperv kvm kvm2 virtualbox vmwarefusion xhyve]")
	cmd.Flags().StringVarP(&options.Flags.HyperVVirtualSwitch, "hyperv-virtual-switch", "v", "", "Additional options for using HyperV with Minikube")
	cmd.Flags().StringVarP(&options.Flags.ClusterVersion, optionKubernetesVersion, "", "", "Kubernetes version")

	return cmd
}

func (o *CreateClusterMinikubeOptions) Run() error {
	var deps []string
	d := opts.BinaryShouldBeInstalled("minikube")
	if d != "" {
		deps = append(deps, d)
	}

	err := o.InstallMissingDependencies(deps)
	if err != nil {
		log.Logger().Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	if o.isExistingMinikubeRunning() {
		log.Logger().Error("an existing Minikube cluster is already running, perhaps use `jx install`.\nNote existing Minikube must have RBAC enabled, running `minikube delete` and `jx create cluster minikube` creates a new VM with RBAC enabled")
		os.Exit(-1)
	}

	err = o.createClusterMinikube()
	if err != nil {
		log.Logger().Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *CreateClusterMinikubeOptions) defaultMacVMDriver() string {
	_, err := o.GetCommandOutput("", "hyperkit", "-v")
	if err != nil {
		log.Logger().Warnf("Could not find hyperkit on your PATH. If you install Docker for Mac then we could use hyperkit.\nSee: https://docs.docker.com/docker-for-mac/install/")
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
		Help:    "Amount of RAM allocated to the Minikube VM in MB",
	}
	showPromptIfOptionNotSet(&mem, prompt, o.In, o.Out, o.Err)

	cpu := o.Flags.CPU
	prompt = &survey.Input{
		Message: "cpu (cores)",
		Default: MinikubeDefaultCpu,
		Help:    "Number of CPUs allocated to the Minikube VM",
	}
	showPromptIfOptionNotSet(&cpu, prompt, o.In, o.Out, o.Err)

	disksize := o.Flags.DiskSize
	prompt = &survey.Input{
		Message: "disk-size (MB)",
		Default: MinikubeDefaultDiskSize,
		Help:    "Total amount of storage allocated to the Minikube VM in MB",
	}
	showPromptIfOptionNotSet(&disksize, prompt, o.In, o.Out, o.Err)

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

	showPromptIfOptionNotSet(&vmDriverValue, prompts, o.In, o.Out, o.Err)

	if vmDriverValue != "none" {
		err := o.DoInstallMissingDependencies([]string{vmDriverValue})
		if err != nil {
			log.Logger().Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
			os.Exit(-1)
		}
	}

	args := []string{"start", "--memory", mem, "--cpus", cpu, "--disk-size", disksize, "--vm-driver", vmDriverValue, "--bootstrapper=kubeadm"}
	// Show verbose output for minikube cluster creation if verbose flag is set
	if o.Verbose {
		args = append(args, "--logtostderr", "--v=3")
	}

	hyperVVirtualSwitch := o.Flags.HyperVVirtualSwitch
	if hyperVVirtualSwitch != "" {
		args = append(args, "--hyperv-virtual-switch", hyperVVirtualSwitch)
	}
	kubernetesVersion := o.Flags.ClusterVersion
	if kubernetesVersion != "" {
		args = append(args, "--kubernetes-version", kubernetesVersion)
	}
	o.Out.Write([]byte("Creating Minikube cluster...\n"))
	err := o.RunCommand("minikube", args...)
	if err != nil {
		return err
	} else {
		o.Out.Write([]byte("Minikube cluster created.\n"))
	}

	err = o.Retry(3, 10*time.Second, func() (err error) {
		err = o.RunCommand("kubectl", "create", "clusterrolebinding", "add-on-cluster-admin", "--clusterrole", "cluster-admin", "--serviceaccount", "kube-system:default")
		return
	})
	if err != nil {
		return err
	}

	if o.CreateClusterOptions.InstallOptions.InitOptions.Flags.Domain == "" {
		ip, err := o.GetCommandOutput("", "minikube", "ip")
		if err != nil {
			return err
		}
		o.InstallOptions.Flags.Domain = ip + ".nip.io"
	} else {
		o.InstallOptions.Flags.Domain = o.CreateClusterOptions.InstallOptions.InitOptions.Flags.Domain
	}

	log.Logger().Info("Initialising cluster ...")
	err = o.initAndInstall(cloud.MINIKUBE)
	if err != nil {
		return err
	}

	context, err := o.GetCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	ns := o.Flags.Namespace
	if ns == "" {
		_, ns, _ = o.KubeClientAndNamespace()
		if err != nil {
			return err
		}
	}

	err = o.RunCommand("kubectl", "config", "set-context", context, "--namespace", ns)
	if err != nil {
		return err
	}

	err = o.RunCommand("kubectl", "get", "ingress")
	if err != nil {
		return err
	}

	return nil
}

func showPromptIfOptionNotSet(option *string, p survey.Prompt, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) error {
	surveyOpts := survey.WithStdio(in, out, errOut)
	if *option == "" {
		err := survey.AskOne(p, option, nil, surveyOpts)
		if err != nil {
			return err
		}
	}
	return nil
}
