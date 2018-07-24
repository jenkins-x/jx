package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const appLabel = "app"

type StepVerifyOptions struct {
	StepOptions

	After       int32
	Application string
	Namespace   string
	Restarts    int32
}

var (
	StepVerifyLong = templates.LongDesc(`
		This pipeline step performs deployment verification
	`)

	StepVerifyExample = templates.Examples(`
		jx step verify
	`)
)

func NewCmdStepVerify(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepVerifyOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "verify",
		Short:   "Performs deployment verification in a pipeline",
		Long:    StepVerifyLong,
		Example: StepVerifyExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().Int32VarP(&options.After, "after", "", 60, "The time in seconds after which the application should be ready")
	cmd.Flags().StringVarP(&options.Application, optionApplication, "a", "", "The Application to verify")
	cmd.Flags().StringVarP(&options.Namespace, kube.OptionNamespace, "n", "", "The Kubernetes namespace where the application to verify runs")
	cmd.Flags().Int32VarP(&options.Restarts, "restarts", "r", 0, "Maximum number of restarts which are acceptable within the given time")

	return cmd
}

func (o *StepVerifyOptions) Run() error {
	var err error
	app := o.Application
	if app == "" {
		app, err = o.DiscoverAppName()
		if err != nil {
			return errors.Wrap(err, "failed to discover application")
		}
	}

	ns := o.Namespace
	if ns == "" {
		ns = o.currentNamespace
	}

	// Wait for the given time to exceed before starting the verification
	time.Sleep(time.Duration(o.After) * time.Second)

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get the Kube client")
	}

	pods, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list the PODs in namespace '%s'", ns)
	}

	foundPod := false
	appNames := []string{app, ns + "-preview", ns + "-" + app}
	for _, pod := range pods.Items {
		appLabelValue, ok := pod.Labels[appLabel]
		if !ok {
			continue
		}
		for _, appName := range appNames {
			if appName == appLabelValue {
				foundPod = true
				restarts := kube.GetPodRestarts(&pod)
				if !kube.IsPodReady(&pod) {
					if restarts < o.Restarts {
						continue
					} else {
						return fmt.Errorf("pod '%s' is '%s' and was restarted '%d', which exceeds max number of restarts '%d'",
							pod.Name, pod.Status.Phase, restarts, o.Restarts)
					}
				} else {
					if restarts > o.Restarts {
						return fmt.Errorf("pod '%s' is running but was restarted '%d' which exceeds max number of restarts '%d'",
							pod.Name, restarts, o.Restarts)
					}
				}
			}
		}
	}

	if !foundPod {
		return fmt.Errorf("no pod found running for application '%s' in namespace '%s'", app, ns)
	}

	return nil
}
