package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LogsOptions struct {
	CommonOptions

	Container       string
	Namespace       string
	Environment     string
	Filter          string
	Label           string
	EditEnvironment bool
	KNativeBuild    bool
}

var (
	logs_long = templates.LongDesc(`
		Tails the logs of the newest pod for a Deployment.

`)

	logs_example = templates.Examples(`
		# Tails the log of the latest pod in deployment myapp
		jx logs myapp

		# Tails the log of the container foo in the latest pod in deployment myapp
		jx logs myapp -c foo

		# Tails the log of the latest Knative build pod
		jx logs -k
`)
)

func NewCmdLogs(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &LogsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "logs [deployment]",
		Short:   "Tails the log of the latest pod for a deployment",
		Long:    logs_long,
		Example: logs_example,
		Aliases: []string{"log"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Container, "container", "c", "", "The name of the container to log")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the Deployment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&options.Environment, "env", "e", "", "the Environment to look for the Deployment. Defaults to the current environment")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters the available deployments if no deployment argument is provided")
	cmd.Flags().StringVarP(&options.Label, "label", "l", "", "The label to filter the pods if no deployment argument is provided")
	cmd.Flags().BoolVarP(&options.KNativeBuild, "knative-build", "k", false, "View the logs of the latest Knative build pod")
	cmd.Flags().BoolVarP(&options.EditEnvironment, "edit", "d", false, "Use my Edit Environment to look for the Deployment pods")
	return cmd
}

func (o *LogsOptions) Run() error {
	args := o.Args

	client, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		env := o.Environment
		if env != "" {
			ns, err = kube.GetEnvironmentNamespace(jxClient, devNs, env)
			if err != nil {
				return err
			}
		}
		if ns == "" && o.EditEnvironment {
			ns, err = kube.GetEditEnvironmentNamespace(jxClient, devNs)
			if err != nil {
				return err
			}
		}
	}
	if ns == "" {
		ns = curNs
	}
	names, err := kube.GetDeploymentNames(client, ns, o.Filter)
	if err != nil {
		return fmt.Errorf("Could not find deployments in namespace %s with filter %s: %s", ns, o.Filter, err)
	}
	if len(names) == 0 {
		if o.Filter == "" {
			return fmt.Errorf("There are no Deployments")
		} else {
			return fmt.Errorf("There are no Deployments matching filter: " + o.Filter)
		}
	}
	name := ""
	if len(args) == 0 {
		if o.Label == "" && !o.KNativeBuild {
			n, err := util.PickName(names, "Pick Deployment:", "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
			name = n
		}
	} else {
		name = args[0]
		if util.StringArrayIndex(names, name) < 0 {
			return util.InvalidArg(name, names)
		}
	}

	for {
		pod := ""
		if o.KNativeBuild {
			pod, err = o.waitForReadyKnativeBuildPod(client, ns, false)
			if pod == "" {
				return fmt.Errorf("No Knative build pod found for namespace %s", ns)
			}
		} else if o.Label != "" {
			selector, err := parseSelector(o.Label)
			if err != nil {
				return err
			}
			pod, err = o.waitForReadyPodForSelectorLabels(client, ns, selector, false)
			if err != nil {
				return err
			}
			if pod == "" {
				return fmt.Errorf("No pod found for namespace %s with selector %s", ns, o.Label)
			}
		} else {
			pod, err = o.waitForReadyPodForDeployment(client, ns, name, names, false)
			if err != nil {
				return err
			}
			if pod == "" {
				return fmt.Errorf("No pod found for namespace %s with name %s", ns, name)
			}
		}
		err = o.tailLogs(ns, pod, o.Container)
		if err != nil {
			return nil
		}
	}
}

func parseSelector(selectorText string) (map[string]string, error) {
	selector, err := metav1.ParseToLabelSelector(selectorText)
	if err != nil {
		return nil, err
	}
	return selector.MatchLabels, nil
}

func (o *CommonOptions) tailLogs(ns string, pod string, containerName string) error {
	args := []string{"logs", "-n", ns, "-f"}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, pod)
	name := "kubectl"
	e := exec.Command(name, args...)
	e.Stderr = o.Err
	stdout, _ := e.StdoutPipe()

	os.Setenv("PATH", util.PathWithBinary())
	err := e.Start()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Fprintln(o.Out, m)
		if m == "Finished: FAILURE" {
			os.Exit(1)
		}
	}
	e.Wait()
	return err
}

// waitForReadyPodForDeployment waits for a ready pod in a Deployment in the given namespace with the given name
func (o *CommonOptions) waitForReadyPodForDeployment(c kubernetes.Interface, ns string, name string, names []string, readyOnly bool) (string, error) {
	deployment, err := c.AppsV1beta1().Deployments(ns).Get(name, metav1.GetOptions{})
	if err != nil || deployment == nil {
		return "", util.InvalidArg(name, names)
	}
	selector := deployment.Spec.Selector
	if selector == nil {
		return "", fmt.Errorf("No selector defined on Deployment %s in namespace %s", name, ns)
	}
	labels := selector.MatchLabels
	if labels == nil {
		return "", fmt.Errorf("No MatchLabels defined on the Selector of Deployment %s in namespace %s", name, ns)
	}
	return o.waitForReadyPodForSelectorLabels(c, ns, labels, readyOnly)
}

func (o *CommonOptions) waitForReadyPodForSelectorLabels(c kubernetes.Interface, ns string, labels map[string]string, readyOnly bool) (string, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	if err != nil {
		return "", err
	}
	return o.waitForReadyPodForSelector(c, ns, selector, readyOnly)
}

func (o *CommonOptions) waitForReadyKnativeBuildPod(c kubernetes.Interface, ns string, readyOnly bool) (string, error) {
	log.Warnf("Waiting for a running Knative build pod in namespace %s\n", ns)
	lastPod := ""
	for {
		pods, err := builds.GetBuildPods(c, ns)
		if err != nil {
			return "", err
		}
		name := ""
		loggedInitContainerIdx := -1
		var latestPod *corev1.Pod
		lastTime := time.Time{}
		for _, pod := range pods {
			phase := pod.Status.Phase
			if phase == corev1.PodRunning || phase == corev1.PodPending {
				if !readyOnly {
					created := pod.CreationTimestamp
					if name == "" || created.After(lastTime) {
						lastTime = created.Time
						name = pod.Name
						latestPod = pod
					}
				}
			}
		}
		if latestPod != nil && name != "" {
			if name != lastPod {
				lastPod = name
				loggedInitContainerIdx = -1
				log.Warnf("Found newest pod: %s\n", name)
			}
			if kube.IsPodReady(latestPod) {
				return name, nil
			}

			initContainers := latestPod.Status.InitContainerStatuses
			for idx, ic := range initContainers {
				if isContainerStarted(&ic.State) && idx > loggedInitContainerIdx {
					loggedInitContainerIdx = idx
					containerName := ic.Name
					log.Warnf("Init container on pod: %s is: %s\n", name, containerName)
					err = o.tailLogs(ns, name, containerName)
					if err != nil {
						break
					}
				}
			}
		}
		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

func (o *CommonOptions) waitForReadyPodForSelector(c kubernetes.Interface, ns string, selector labels.Selector, readyOnly bool) (string, error) {
	log.Warnf("Waiting for a running pod in namespace %s with labels %v\n", ns, selector.String())
	lastPod := ""
	for {
		pods, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return "", err
		}
		name := ""
		loggedInitContainerIdx := -1
		var latestPod *corev1.Pod
		lastTime := time.Time{}
		for _, pod := range pods.Items {
			phase := pod.Status.Phase
			if phase == corev1.PodRunning || phase == corev1.PodPending {
				if !readyOnly {
					created := pod.CreationTimestamp
					if name == "" || created.After(lastTime) {
						lastTime = created.Time
						name = pod.Name
						latestPod = &pod
					}
				}
			}
		}
		if latestPod != nil && name != "" {
			if name != lastPod {
				lastPod = name
				loggedInitContainerIdx = -1
				log.Warnf("Found newest pod: %s\n", name)
			}
			if kube.IsPodReady(latestPod) {
				return name, nil
			}

			initContainers := latestPod.Status.InitContainerStatuses
			for idx, ic := range initContainers {
				if isContainerStarted(&ic.State) && idx > loggedInitContainerIdx {
					loggedInitContainerIdx = idx
					containerName := ic.Name
					log.Warnf("Init container on pod: %s is: %s\n", name, containerName)
					err = o.tailLogs(ns, name, containerName)
					if err != nil {
						break
					}
				}
			}
		}
		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

func isContainerStarted(state *corev1.ContainerState) bool {
	if state == nil {
		return false
	}
	if state.Running != nil {
		return !state.Running.StartedAt.IsZero()
	}
	if state != nil && state.Terminated != nil {
		return !state.Terminated.StartedAt.IsZero()
	}
	return false
}

func isContainerCompleted(state *corev1.ContainerState) bool {
	if state != nil && state.Terminated != nil {
		return !state.Terminated.FinishedAt.IsZero()
	}
	return false
}
