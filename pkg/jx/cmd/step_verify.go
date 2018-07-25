package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
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
	Pods        int32
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
	cmd.Flags().Int32VarP(&options.Pods, "pods", "p", 1, "Number of expected pods to be running")
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

	log.Infof("Verifying if app '%s' is running in namespace '%s'", app, ns)

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get the Kube client")
	}

	pods, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list the PODs in namespace '%s'", ns)
	}

	var foundPods int32
	appNames := []string{app, ns + "-preview", ns + "-" + app}
	for _, pod := range pods.Items {
		appLabelValue, ok := pod.Labels[appLabel]
		if !ok {
			continue
		}
		for _, appName := range appNames {
			if appName == appLabelValue {
				foundPods += 1
				restarts := kube.GetPodRestarts(&pod)
				if !kube.IsPodReady(&pod) {
					if restarts < o.Restarts {
						continue
					} else {
						err = o.updatePipelineActivity(v1.ActivityStatusTypeFailed)
						if err != nil {
							return err
						}
						return fmt.Errorf("pod '%s' is '%s' and was restarted '%d', which exceeds max number of restarts '%d'",
							pod.Name, pod.Status.Phase, restarts, o.Restarts)
					}
				} else {
					if restarts > o.Restarts {
						err = o.updatePipelineActivity(v1.ActivityStatusTypeFailed)
						if err != nil {
							return err
						}
						return fmt.Errorf("pod '%s' is running but was restarted '%d', which exceeds max number of restarts '%d'",
							pod.Name, restarts, o.Restarts)
					}
				}
			}
		}
	}

	if foundPods != o.Pods {
		err = o.updatePipelineActivity(v1.ActivityStatusTypeFailed)
		if err != nil {
			return err
		}
		return fmt.Errorf("found '%d' pods running but expects '%d'", foundPods, o.Pods)
	}

	err = o.updatePipelineActivity(v1.ActivityStatusTypeSucceeded)
	if err != nil {
		return err
	}
	return nil
}

func (o *StepVerifyOptions) updatePipelineActivity(status v1.ActivityStatusType) error {
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the api extensions client")
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the pipeline activity CRD")
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to get the jx client")
	}

	pipeline := os.Getenv("JOB_NAME")
	build := os.Getenv("BUILD_NUMBER")
	if pipeline != "" && build != "" {
		name := kube.ToValidName(pipeline + "-" + build)
		activities := jxClient.JenkinsV1().PipelineActivities(devNs)
		activity, err := activities.Get(name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get the activity with name '%s'", name)
		}
		activity.Spec.Status = status
		_, err = activities.Update(activity)
		if err != nil {
			return errors.Wrapf(err, "failed to update activity status to '%s'", status)
		}
	}

	return nil
}
