package cmd

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type LogsOptions struct {
	CommonOptions

	Container string
	Namespace string
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
	return cmd
}

func (o *LogsOptions) Run() error {
	args := o.Args

	client, curNs, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = curNs
	}
	name := ""
	if len(args) == 0 {
		names, err := GetDeploymentNames(client, ns)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return fmt.Errorf("There are no Deployments running")
		}
		n, err := util.PickName(names, "Pick Deployment:")
		if err != nil {
			return err
		}
		name = n
	} else {
		name = args[0]
	}

	for {
		pod, err := waitForReadyPodForDeployment(client, ns, name)
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
		err = o.runCommand("kubectl", args...)
		if err != nil {
			return nil
		}
	}
}

// waitForReadyPodForDeployment waits for a ready pod in a Deployment in the given namespace with the given name
func waitForReadyPodForDeployment(c *kubernetes.Clientset, ns string, name string) (string, error) {
	deployment, err := c.AppsV1beta2().Deployments(ns).Get(name, metav1.GetOptions{})
	if err != nil || deployment == nil {
		names, e2 := GetDeploymentNames(c, ns)
		if e2 == nil {
			return "", util.InvalidArg(name, names)
		}
		return "", fmt.Errorf("Could not find a Deployment %s in namespace %s due to %v", name, ns, err)
	}
	selector := deployment.Spec.Selector
	if selector == nil {
		return "", fmt.Errorf("No selector defined on Deployment %s in namespace %s", name, ns)
	}
	labels := selector.MatchLabels
	if labels == nil {
		return "", fmt.Errorf("No MatchLabels defined on the Selector of Deployment %s in namespace %s", name, ns)
	}
	return waitForReadyPodForSelector(c, ns, labels)
}

func waitForReadyPodForSelector(c *kubernetes.Clientset, ns string, labels map[string]string) (string, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	if err != nil {
		return "", err
	}
	fmt.Printf(util.ColorStatus("Waiting for a running pod in namespace %s with labels %v\n"), ns, labels)
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
				created := pod.CreationTimestamp
				if name == "" || created.After(lastTime) {
					lastTime = created.Time
					name = pod.Name
				}
			}
		}
		if name != "" {
			fmt.Printf(util.ColorStatus("Found newest pod: %s\n"), util.ColorInfo(name))
			return name, nil
		}
		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

// TODO move to kube/deployments.go when jrawlings merges his stuff install...
func GetDeploymentNames(client *kubernetes.Clientset, ns string) ([]string, error) {
	names := []string{}
	list, err := client.AppsV1beta2().Deployments(ns).List(metav1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Deployments %s", err)
	}
	for _, n := range list.Items {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	return names, nil
}
