package cmd

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
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

	After    int32
	Pods     int32
	Restarts int32
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
	cmd.Flags().Int32VarP(&options.Pods, "pods", "p", 1, "Number of expected pods to be running")
	cmd.Flags().Int32VarP(&options.Restarts, "restarts", "r", 0, "Maximum number of restarts which are acceptable within the given time")

	return cmd
}

func (o *StepVerifyOptions) Run() error {
	activity, err := o.detectPipelineActivity()
	if err != nil {
		return errors.Wrap(err, "failed to detect the pipeline activity")
	}

	app, ns, err := o.determineAppAndNamespace(activity)
	if err != nil {
		return errors.Wrap(err, "failed to determine the application name and namespace from pipeline activity")
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
						err = o.updatePipelineActivity(activity, v1.ActivityStatusTypeFailed)
						if err != nil {
							return err
						}
						return fmt.Errorf("pod '%s' is '%s' and was restarted '%d', which exceeds max number of restarts '%d'",
							pod.Name, pod.Status.Phase, restarts, o.Restarts)
					}
				} else {
					if restarts > o.Restarts {
						err = o.updatePipelineActivity(activity, v1.ActivityStatusTypeFailed)
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
		err = o.updatePipelineActivity(activity, v1.ActivityStatusTypeFailed)
		if err != nil {
			return err
		}
		return fmt.Errorf("found '%d' pods running but expects '%d'", foundPods, o.Pods)
	}

	err = o.updatePipelineActivity(activity, v1.ActivityStatusTypeSucceeded)
	if err != nil {
		return err
	}
	return nil
}

func (o *StepVerifyOptions) detectPipelineActivity() (*v1.PipelineActivity, error) {
	pipeline := os.Getenv("JOB_NAME")
	build := os.Getenv("BUILD_NUMBER")
	if pipeline == "" || build == "" {
		return nil, errors.New("JOB_NAME or BUILD_NUMBER environment variables not set")
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the api extensions client")
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to register the pipeline activity CRD")
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the jx client")
	}

	name := kube.ToValidName(pipeline + "-" + build)
	activities := jxClient.JenkinsV1().PipelineActivities(devNs)
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the activity with name '%s'", name)
	}
	return activity, nil
}

func (o *StepVerifyOptions) determineAppAndNamespace(activity *v1.PipelineActivity) (string, string, error) {
	for _, step := range activity.Spec.Steps {
		if step.Kind == v1.ActivityStepKindTypePreview {
			preview := step.Preview
			if preview == nil {
				return "", "", fmt.Errorf("empty preview step in pipeline activity '%s'", activity.Name)
			}
			applicationURL := preview.ApplicationURL
			if applicationURL == "" {
				return "", "", fmt.Errorf("empty application URL in pipeline activity '%s'", activity.Name)
			}
			return o.parseApplicationURL(applicationURL)
		}

		if step.Kind == v1.ActivityStepKindTypePromote {
			promote := step.Promote
			if promote != nil {
				return "", "", fmt.Errorf("empty promote step in pipeline activity '%s'", activity.Name)
			}
			applicationURL := promote.ApplicationURL
			if applicationURL == "" {
				return "", "", fmt.Errorf("empty application URL in pipeline activity '%s'", activity.Name)
			}
			return o.parseApplicationURL(applicationURL)
		}
	}
	return "", "", fmt.Errorf("could not determine the application name and namespace from activity '%s'", activity.Name)
}

func (o *StepVerifyOptions) parseApplicationURL(applicationURL string) (string, string, error) {
	url, err := url.Parse(applicationURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse the application URL '%s'", applicationURL)
	}

	host, _, _ := net.SplitHostPort(url.Host)
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot parse the application name and namespace from URL Host '%s'", host)
	}

	return parts[0], parts[1], nil
}

func (o *StepVerifyOptions) updatePipelineActivity(activity *v1.PipelineActivity, status v1.ActivityStatusType) error {
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

	activities := jxClient.JenkinsV1().PipelineActivities(devNs)
	activity.Spec.Status = status
	_, err = activities.Update(activity)
	if err != nil {
		return errors.Wrapf(err, "failed to update activity status to '%s'", status)
	}

	return nil
}
