package cmd

import (
	"fmt"
	"io"
	"os/user"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	DefaultShell = "/bin/sh"
)

type RshOptions struct {
	CommonOptions

	Container  string
	Namespace  string
	Pod        string
	Executable string
	DevPod     bool

	stopCh chan struct{}
}

var (
	rsh_long = templates.LongDesc(`
		Opens a terminal or runs a command in a pods container

`)

	rsh_example = templates.Examples(`
		# Open a terminal in the first container of the foo deployment's latest pod
		jx rsh foo

		# Opens a terminal in the cheese container in the latest pod in the foo deployment
		jx rsh -c cheese foo

		# To connect to one of your DevPods use:
		jx rsh -d
`)
)

func NewCmdRsh(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &RshOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "rsh [deploymentOrPodName]",
		Short:   "Opens a terminal in a pod or runs a command in the pod",
		Long:    rsh_long,
		Example: rsh_example,
		Aliases: []string{"log"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Container, "container", "c", "", "The name of the container to log")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the Deployment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&options.Namespace, "pod", "p", "", "the pod name to use")
	cmd.Flags().StringVarP(&options.Executable, "shell", "s", "", "Path to the shell command")
	cmd.Flags().BoolVarP(&options.DevPod, "devpod", "d", false, "Connect to a DevPod")
	return cmd
}

func (o *RshOptions) Run() error {
	args := o.Args

	client, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = curNs
	}

	filter := ""
	names := []string{}
	podsName := "Pods"
	pods := map[string]*corev1.Pod{}
	if o.DevPod {
		podsName = "DevPods"
		u, err := user.Current()
		if err != nil {
			return err
		}
		names, pods, err = kube.GetDevPodNames(client, ns, u.Username)
		if err != nil {
			return err
		}
	} else {
		if o.Executable == "" {
			o.Executable = DefaultShell
		}
		names, err = kube.GetPodNames(client, ns, "")
		if err != nil {
			return err
		}
	}
	if len(names) == 0 {
		if filter == "" {
			return fmt.Errorf("There are no %s", podsName)
		} else {
			return fmt.Errorf("There are no %s matching filter: %s", podsName, filter)
		}
	}
	name := o.Pod
	if len(args) == 0 {
		if util.StringArrayIndex(names, name) < 0 {
			n, err := util.PickName(names, "Pick Pod:")
			if err != nil {
				return err
			}
			name = n
		}
	} else {
		name = args[0]
		if util.StringArrayIndex(names, name) < 0 {
			// lets try use the name as a filter
			filteredNames := []string{}
			for _, n := range names {
				if strings.Contains(n, name) {
					filteredNames = append(filteredNames, n)
				}
			}
			n, err := util.PickName(filteredNames, "Pick Pod:")
			if err != nil {
				return err
			}
			name = n
		}
	}

	if name == "" {
		return fmt.Errorf("No pod found for namespace %s with name %s", ns, name)
	}

	commandArguments := []string{}
	if o.Executable != "" {
		commandArguments = []string{o.Executable}
	}

	if o.DevPod {
		if o.Executable == "" {
			workingDir := ""
			pod := pods[name]
			if pod != nil && pod.Annotations != nil {
				workingDir = pod.Annotations[kube.AnnotationWorkingDir]
			}
			if workingDir != "" {
				commandArguments = []string{"--", "/bin/sh", "-c", "mkdir -p " + workingDir + "\ncd " + workingDir + "\nbash"}
			} else {
				commandArguments = []string{"--", "/bin/sh", "-c", "bash"}
			}
		}
	}

	a := []string{"exec", "-it", "-n", ns}
	if o.Container != "" {
		a = append(a, "-c", o.Container)
	}
	a = append(a, name)
	if len(args) > 1 {
		a = append(a, args[1:]...)
	} else if len(commandArguments) > 0 {
		a = append(a, commandArguments...)
	}
	return o.runCommandInteractive(true, "kubectl", a...)
}
