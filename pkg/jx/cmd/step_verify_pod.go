package cmd

import (
	"bytes"
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
)

var (
	stepStatusLong = templates.LongDesc(`
		This step checks the status of all kubernetes pods
	`)

	stepStatusExample = templates.Examples(`
		jx step verify pod
	`)
)

type StepVerifyPodOptions struct {
	StepOptions
	Debug bool
}

// NewCmdStepVerifyPod creates the `jx step verify pod` command
func NewCmdStepVerifyPod(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {

	options := StepVerifyPodOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pod",
		Short:   "status of kubernetes pods",
		Long:    stepStatusLong,
		Example: stepStatusExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Debug, "debug", "", false, "Output logs of any failed pod")

	return cmd
}

// Run the `jx step verify pod` command
func (o *StepVerifyPodOptions) Run() error {
	kubeClient, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to get the Kube client")
	}

	pods, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list the PODs in namespace '%s'", ns)
	}

	fmt.Println("Checking pod statuses")

	table := o.createTable()
	table.AddRow("POD", "STATUS")

	var f *os.File

	if o.Debug {
		fmt.Println("Creating verify-pod.log file")
		f, err = os.Create("verify-pod.log")
		if err != nil {
			return errors.Wrap(err, "error creating log file")
		}
		defer f.Close()
	}

	for _, pod := range pods.Items {
		podName := pod.ObjectMeta.Name
		phase := pod.Status.Phase

		if phase == corev1.PodFailed && o.Debug {
			args := []string{"logs", podName}
			name := "kubectl"
			e := exec.Command(name, args...)
			e.Stderr = o.Err
			var out bytes.Buffer
			e.Stdout = &out
			err := e.Run()
			if err != nil {
				return errors.Wrap(err, "failed to get the Kube pod logs")
			}
			_, err = f.WriteString(fmt.Sprintf("Logs for pod %s:\n", podName))
			if err != nil {
				return errors.Wrap(err, "error writing log file")
			}
			_, err = f.Write(out.Bytes())
			if err != nil {
				return errors.Wrap(err, "error writing log file")
			}
		}
		table.AddRow(podName, string(phase))
	}
	table.Render()
	return nil
}
