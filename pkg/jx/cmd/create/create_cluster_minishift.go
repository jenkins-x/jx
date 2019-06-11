package create

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/features"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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
		This command creates a new Kubernetes cluster, installing required local dependencies and provisions the
		Jenkins X platform

		Minishift is a tool that makes it easy to run OpenShift locally. Minishift runs a single-node OpenShift
		cluster inside a VM on your laptop for users looking to try out Kubernetes or develop with it day-to-day.

`)

	createClusterMinishiftExample = templates.Examples(`

		jx create cluster minishift

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdCreateClusterMinishift(commonOpts *opts.CommonOptions) *cobra.Command {
	options := CreateClusterMinishiftOptions{
		CreateClusterOptions: createCreateClusterOptions(commonOpts, cloud.MINISHIFT),
	}
	cmd := &cobra.Command{
		Use:     "minishift",
		Short:   "Create a new OpenShift cluster with Minishift: Runs locally",
		Long:    createClusterMinishiftLong,
		Example: createClusterMinishiftExample,
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

	cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "4096", "Amount of RAM allocated to the Minishift VM in MB")
	cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "3", "Number of CPUs allocated to the Minishift VM")
	cmd.Flags().StringVarP(&options.Flags.Driver, "vm-driver", "d", "", "VM driver is one of: [virtualbox xhyve vmwarefusion hyperkit]")
	cmd.Flags().StringVarP(&options.Flags.HyperVVirtualSwitch, "hyperv-virtual-switch", "v", "", "Additional options for using HyperV with Minishift")

	return cmd
}

func (o *CreateClusterMinishiftOptions) Run() error {
	var deps []string
	d := opts.BinaryShouldBeInstalled("minishift")
	if d != "" {
		deps = append(deps, d)
	}
	d = opts.BinaryShouldBeInstalled("oc")
	if d != "" {
		deps = append(deps, d)
	}

	err := o.InstallMissingDependencies(deps)
	if err != nil {
		log.Logger().Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	if o.isExistingMinishiftRunning() {
		log.Logger().Error("an existing Minishift cluster is already running, perhaps use `jx install`.\nNote: existing Minishift must have RBAC enabled, running `minishift delete` and `jx create cluster minishift` creates a new VM with RBAC enabled")
		os.Exit(-1)
	}

	err = o.createClusterMinishift()
	if err != nil {
		log.Logger().Errorf("error creating cluster %v", err)
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
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	mem := o.Flags.Memory
	prompt := &survey.Input{
		Message: "memory (MB)",
		Default: mem,
		Help:    "Amount of RAM allocated to the Minishift VM in MB",
	}
	survey.AskOne(prompt, &mem, nil, surveyOpts)

	cpu := o.Flags.CPU
	prompt = &survey.Input{
		Message: "cpu (cores)",
		Default: cpu,
		Help:    "Number of CPUs allocated to the Minishift VM",
	}
	survey.AskOne(prompt, &cpu, nil, surveyOpts)

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

	err := survey.AskOne(prompts, &driver, nil, surveyOpts)
	if err != nil {
		return err
	}

	if driver != "none" {
		err = o.DoInstallMissingDependencies([]string{driver})
		if err != nil {
			log.Logger().Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
			os.Exit(-1)
		}
	}

	log.Logger().Info("Installing default addons ...")
	err = o.RunCommand("minishift", "addons", "install", "--defaults")
	if err != nil {
		return err
	}

	log.Logger().Info("Enabling admin user...")
	err = o.RunCommand("minishift", "addons", "enable", "admin-user")
	if err != nil {
		return err
	}

	args := []string{"start", "--memory", mem, "--cpus", cpu, "--vm-driver", driver}
	hyperVVirtualSwitch := o.Flags.HyperVVirtualSwitch
	if hyperVVirtualSwitch != "" {
		args = append(args, "--hyperv-virtual-switch", hyperVVirtualSwitch)
	}

	log.Logger().Info("Creating cluster...")
	err = o.RunCommand("minishift", args...)
	if err != nil {
		return err
	}

	ip, err := o.GetCommandOutput("", "minishift", "ip")
	if err != nil {
		return err
	}
	o.InstallOptions.Flags.Domain = ip + ".nip.io"

	err = o.RunCommand("oc", "login", "-u", "admin", "-p", "admin", "--insecure-skip-tls-verify=true")
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

	log.Logger().Info("Initialising cluster ...")
	err = o.initAndInstall(cloud.MINISHIFT)
	if err != nil {
		return err
	}

	return nil
}
