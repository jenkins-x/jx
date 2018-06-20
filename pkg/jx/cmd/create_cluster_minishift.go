package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// CreateClusterMinishiftOptions the flags for running create cluster
type CreateClusterMinishiftOptions struct {
	CreateClusterOptions

	Flags    CreateClusterMinishiftFlags
	Provider KubernetesProvider
}

type CreateClusterMinishiftFlags struct {
	Memory              string
	CPU                 string
	Driver              string
	HyperVVirtualSwitch string
	Namespace           string
}

var (
	createClusterMinishiftLong = templates.LongDesc(`
		This command creates a new kubernetes cluster, installing required local dependencies and provisions the
		Jenkins X platform

		Minishift is a tool that makes it easy to run OpenShift locally. Minishift runs a single-node OpenShift
		cluster inside a VM on your laptop for users looking to try out Kubernetes or develop with it day-to-day.

`)

	createClusterMinishiftExample = templates.Examples(`

		jx create cluster minishift

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateClusterMinishift(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterMinishiftOptions{
		CreateClusterOptions: createCreateClusterOptions(f, out, errOut, MINISHIFT),
	}
	cmd := &cobra.Command{
		Use:     "minishift",
		Short:   "Create a new OpenShift cluster with minishift: Runs locally",
		Long:    createClusterMinishiftLong,
		Example: createClusterMinishiftExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "4096", "Amount of RAM allocated to the minishift VM in MB")
	cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "3", "Number of CPUs allocated to the minishift VM")
	cmd.Flags().StringVarP(&options.Flags.Driver, "vm-driver", "d", "", "VM driver is one of: [virtualbox xhyve vmwarefusion hyperkit]")
	cmd.Flags().StringVarP(&options.Flags.HyperVVirtualSwitch, "hyperv-virtual-switch", "v", "", "Additional options for using HyperV with minishift")

	return cmd
}

func (o *CreateClusterMinishiftOptions) Run() error {
	var deps []string
	d := binaryShouldBeInstalled("minishift")
	if d != "" {
		deps = append(deps, d)
	}
	d = binaryShouldBeInstalled("oc")
	if d != "" {
		deps = append(deps, d)
	}

	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	if o.isExistingMinishiftRunning() {
		log.Error("an existing minishift cluster is already running, perhaps use `jx install`.\nNote existing minishift musty have RBAC enabled, running `minishift delete` and `jx create cluster minishift` creates a new VM with RBAC enabled")
		os.Exit(-1)
	}

	err = o.createClusterMinishift()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *CreateClusterMinishiftOptions) defaultMacVMDriver() string {
	return "xhyve"
}

func (o *CreateClusterMinishiftOptions) isExistingMinishiftRunning() bool {
	var cmd_out bytes.Buffer

	e := exec.Command("minishift", "status")
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

func (o *CreateClusterMinishiftOptions) createClusterMinishift() error {
	mem := o.Flags.Memory
	prompt := &survey.Input{
		Message: "memory (MB)",
		Default: mem,
		Help:    "Amount of RAM allocated to the minishift VM in MB",
	}
	survey.AskOne(prompt, &mem, nil)

	cpu := o.Flags.CPU
	prompt = &survey.Input{
		Message: "cpu (cores)",
		Default: cpu,
		Help:    "Number of CPUs allocated to the minishift VM",
	}
	survey.AskOne(prompt, &cpu, nil)

	vmDriverValue := o.Flags.Driver

	if len(vmDriverValue) == 0 {
		switch runtime.GOOS {
		case "darwin":
			vmDriverValue = o.defaultMacVMDriver()
		case "windows":
			vmDriverValue = "hyperv"
		case "linux":
			vmDriverValue = "kvm"
		default:
			vmDriverValue = "virtualbox"
		}
	}

	// only add drivers that are appropriate for this OS
	var driver string
	drivers := []string{vmDriverValue}
	if vmDriverValue != "virtualbox" {
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
		Default: vmDriverValue,
		Help:    "VM driver, defaults to recommended native virtualisation",
	}

	err := survey.AskOne(prompts, &driver, nil)
	if err != nil {
		return err
	}

	if driver != "none" {
		err = o.doInstallMissingDependencies([]string{driver})
		if err != nil {
			log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
			os.Exit(-1)
		}
	}

	log.Info("Installing default addons ...\n")
	err = o.runCommand("minishift", "addons", "install", "--defaults")
	if err != nil {
		return err
	}

	log.Info("Enabling admin user...\n")
	err = o.runCommand("minishift", "addons", "enable", "admin-user")
	if err != nil {
		return err
	}

	args := []string{"start", "--memory", mem, "--cpus", cpu, "--vm-driver", driver}
	hyperVVirtualSwitch := o.Flags.HyperVVirtualSwitch
	if hyperVVirtualSwitch != "" {
		args = append(args, "--hyperv-virtual-switch", hyperVVirtualSwitch)
	}

	log.Info("Creating cluster...\n")
	err = o.runCommand("minishift", args...)
	if err != nil {
		return err
	}

	ip, err := o.getCommandOutput("", "minishift", "ip")
	if err != nil {
		return err
	}
	o.InstallOptions.Flags.Domain = ip + ".nip.io"

	err = o.runCommand("oc", "login", "-u", "admin", "-p", "admin", "--insecure-skip-tls-verify=true")
	if err != nil {
		return err
	}

	ns := o.Flags.Namespace
	if ns == "" {
		_, ns, _ = o.KubeClient()
		if err != nil {
			return err
		}
	}

	log.Info("Initialising cluster ...\n")
	err = o.initAndInstall(MINISHIFT)
	if err != nil {
		return err
	}

	return nil
}
