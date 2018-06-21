package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/client-go/kubernetes"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LogsOptions struct {
	CommonOptions

	Container       string
	Namespace       string
	Environment     string
	Filter          string
	EditEnvironment bool
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
`)
)

func NewCmdLogs(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &LogsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
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
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Container, "container", "c", "", "The name of the container to log")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the Deployment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&options.Environment, "env", "e", "", "the Environment to look for the Deployment. Defaults to the current environment")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters the available deployments if no deployment argument is provided")
	cmd.Flags().BoolVarP(&options.EditEnvironment, "edit", "d", false, "Use my Edit Environment to look for the Deployment pods")
	return cmd
}

func (o *LogsOptions) Run() error {
	args := o.Args

	client, curNs, err := o.KubeClient()
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
		n, err := util.PickName(names, "Pick Deployment:")
		if err != nil {
			return err
		}
		name = n
	} else {
		name = args[0]
		if util.StringArrayIndex(names, name) < 0 {
			return util.InvalidArg(name, names)
		}
	}

	for {
		pod, err := waitForReadyPodForDeployment(client, ns, name, names, false)
		if err != nil {
			return err
		}
		if pod == "" {
			return fmt.Errorf("No pod found for namespace %s with name %s", ns, name)
		}
		args := []string{"logs", "-n", ns, "-f"}
		if o.Container != "" {
			args = append(args, "-c", o.Container)
		}
		args = append(args, pod)
		o.Verbose = true
		err = o.runCommand("kubectl", args...)
		if err != nil {
			return nil
		}
	}
}

// waitForReadyPodForDeployment waits for a ready pod in a Deployment in the given namespace with the given name
func waitForReadyPodForDeployment(c kubernetes.Interface, ns string, name string, names []string, readyOnly bool) (string, error) {
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
	return waitForReadyPodForSelector(c, ns, labels, readyOnly)
}

func waitForReadyPodForSelector(c kubernetes.Interface, ns string, labels map[string]string, readyOnly bool) (string, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	if err != nil {
		return "", err
	}
	log.Warnf("Waiting for a running pod in namespace %s with labels %v\n", ns, labels)
	for {
		pods, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return "", err
		}
		name := ""
		lastTime := time.Time{}
		for _, pod := range pods.Items {
			phase := pod.Status.Phase
			if phase == corev1.PodRunning {
				if !readyOnly || kube.IsPodReady(&pod) {
					created := pod.CreationTimestamp
					if name == "" || created.After(lastTime) {
						lastTime = created.Time
						name = pod.Name
					}
				}
			}
		}
		if name != "" {
			log.Warnf("Found newest pod: %s\n", name)
			return name, nil
		}
		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}
