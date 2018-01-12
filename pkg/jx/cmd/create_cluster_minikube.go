package cmd

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"runtime"

	"bytes"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// CreateClusterOptions the flags for running crest cluster
type CreateClusterMinikubeOptions struct {
	CreateClusterOptions

	Flags    CreateClusterMinikubeFlags
	Provider KubernetesProvider
}

type CreateClusterMinikubeFlags struct {
	Memory string
	CPU    string
	Driver string
}

var (
	createClusterMinikubeLong = templates.LongDesc(`
		This command creates a new kubernetes cluster, installing required local dependencies and provisions the
		Jenkins-X platform

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
		CreateClusterOptions: CreateClusterOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
			Provider: MINIKUBE,
		},
	}
	cmd := &cobra.Command{
		Use:     "minikube",
		Short:   "Create a new kubernetes cluster with minikube",
		Long:    createClusterMinikubeLong,
		Example: createClusterMinikubeExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Flags.Memory, "memory", "m", "4096", "Amount of RAM allocated to the minikube VM in MB")
	cmd.Flags().StringVarP(&options.Flags.CPU, "cpu", "c", "3", "Number of CPUs allocated to the minikube VM")
	cmd.Flags().StringVarP(&options.Flags.Driver, "vm-driver", "d", "", "VM driver is one of: [virtualbox xhyve vmwarefusion hyperkit]")

	return cmd
}

func (o *CreateClusterMinikubeOptions) Run() error {

	var deps []string
	d := getDependenciesToInstall("minikube")
	if d != "" {
		deps = append(deps, d)
	}

	// Platform specific deps
	if runtime.GOOS == "darwin" {
		d = getDependenciesToInstall("hyperkit")
		if d != "" {
			deps = append(deps, d)
		}
	}

	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("error installing missing dependencies %v, please fix or install manually then try again", err)
		os.Exit(-1)
	}

	if o.isExistingMinikubeRunning() {
		log.Error("an existing minikube cluster is already running, perhaps use jx install")
		os.Exit(-1)
	}

	err = o.createClusterMinikube()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		os.Exit(-1)
	}

	return nil
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
		Default: mem,
		Help:    "Amount of RAM allocated to the minikube VM in MB",
	}
	survey.AskOne(prompt, &mem, nil)

	cpu := o.Flags.CPU
	prompt = &survey.Input{
		Message: "cpu (cores)",
		Default: cpu,
		Help:    "Number of CPUs allocated to the minikube VM",
	}
	survey.AskOne(prompt, &cpu, nil)

	vmDriverValue := o.Flags.Driver

	if len(vmDriverValue) == 0 {
		switch runtime.GOOS {
		case "darwin":
			vmDriverValue = "hyperkit"
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

	prompts := &survey.Select{
		Message: "Select driver:",
		Options: drivers,
		Default: vmDriverValue,
		Help:    "VM driver, defaults to recommended native virtualisation",
	}

	survey.AskOne(prompts, &driver, nil)

	err := o.runCommand("minikube", "start", "--memory", mem, "--cpus", cpu, "--vm-driver", driver)
	if err != nil {
		return err
	}
	return nil
}
