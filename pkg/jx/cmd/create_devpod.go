package cmd

import (
	"fmt"
	"io"
	"os/user"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	optionLabel = "label"
)

var (
	createDevPodLong = templates.LongDesc(`
		Creates a new DevPod

		For more documentation see: [http://jenkins-x.io/developing/devpods/](http://jenkins-x.io/developing/devpods/)

`)

	createDevPodExample = templates.Examples(`
		# creates a new DevPod asking the user for the label to use
		jx create devpod

		# creates a new Maven DevPod 
		jx create devpod -l maven
	`)
)

// CreateDevPodOptions the options for the create spring command
type CreateDevPodOptions struct {
	CreateOptions

	Label      string
	Suffix     string
	WorkingDir string
}

// NewCmdCreateDevPod creates a command object for the "create" command
func NewCmdCreateDevPod(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateDevPodOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "devpod",
		Short:   "Creates a Developer Pod for running builds and tests inside the cluster",
		Aliases: []string{"dpod", "buildpod"},
		Long:    createDevPodLong,
		Example: createDevPodExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Label, optionLabel, "l", "", "The label of the pod template to use")
	cmd.Flags().StringVarP(&options.Suffix, "suffix", "s", "", "The suffix to append the pod name")
	cmd.Flags().StringVarP(&options.WorkingDir, "working-dir", "w", "", "The working directory of the dev pod")
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *CreateDevPodOptions) Run() error {
	client, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(client, curNs)
	if err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps(ns).Get(kube.ConfigMapJenkinsPodTemplates, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Failed to find ConfigMap %s in namespace %s: %s", kube.ConfigMapJenkinsPodTemplates, ns, err)
	}
	podTemplates := cm.Data
	labels := util.SortedMapKeys(podTemplates)

	label := o.Label
	yml := ""
	if label != "" {
		yml = podTemplates[label]
		if yml == "" {
			return util.InvalidOption(optionLabel, label, labels)
		}
	}
	if label == "" {
		label, err = util.PickName(labels, "Pick which kind of dev pod you wish to create: ")
		if err != nil {
			return err
		}
		yml = podTemplates[label]
		if yml == "" {
			return fmt.Errorf("Could not find YAML for pod template label %s", label)
		}
	}
	o.Printf("Creating a dev pod of label: %s\n", label)

	pod := &corev1.Pod{}
	err = yaml.Unmarshal([]byte(yml), &pod)
	if err != nil {
		return fmt.Errorf("Failed to parse Pod Template YAML: %s\n%s", err, yml)
	}

	u, err := user.Current()
	if err != nil {
		return err
	}

	userName := u.Username
	name := kube.ToValidName(userName + "-" + label)
	if o.Suffix != "" {
		name += "-" + o.Suffix
	}
	names, err := kube.GetPodNames(client, ns, "")
	if err != nil {
		return err
	}

	name = uniquePodName(names, name)

	pod.Name = name
	pod.Labels[kube.LabelPodTemplate] = label
	pod.Labels[kube.LabelDevPodName] = name
	pod.Labels[kube.LabelDevPodUsername] = userName

	container1 := &pod.Spec.Containers[0]
	workingDir := o.WorkingDir
	if workingDir == "" {
		workingDir = "/code"
		if label == "go" {
			// TODO add org + repo name
			workingDir = " /home/jenkins/go/src/github.com"
		}
	}
	container1.WorkingDir = workingDir

	_, err = client.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		if o.Verbose {
			return fmt.Errorf("Failed to create pod %s\nYAML: %s", err, yml)
		} else {
			return fmt.Errorf("Failed to create pod %s", err)
		}
	}

	o.Printf("Created pod %s - waiting for it to be ready...\n", util.ColorInfo(name))
	err = kube.WaitForPodNameToBeReady(client, ns, name, time.Hour)
	if err != nil {
		return err
	}

	o.Printf("Pod %s is now ready!\n", util.ColorInfo(name))
	o.Printf("You can open other shells into this DevPod via %s\n", util.ColorInfo("jx rsh -d"))

	options := &RshOptions{
		CommonOptions: o.CommonOptions,
		Namespace:     ns,
		Executable:    "bash",
		Pod:           name,
	}
	options.Args = []string{}
	return options.Run()
}

func uniquePodName(names []string, prefix string) string {
	count := 1
	for {
		name := prefix
		if count > 1 {
			name += strconv.Itoa(count)
		}
		if util.StringArrayIndex(names, name) < 0 {
			return name
		}
		count++
	}
}
