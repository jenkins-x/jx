package cmd

import (
	"io"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/kube"
)

// ControllerBuildOptions are the flags for the commands
type ControllerBuildOptions struct {
	ControllerOptions

	Namespace string
}

// NewCmdControllerBuild creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerBuild(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ControllerBuildOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Runs the buid controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"builds"},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to watch or defaults to the current namespace")
	return cmd
}

// Run implements this command
func (o *ControllerBuildOptions) Run() error {
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = devNs
	}
	pod := &corev1.Pod{}
	log.Infof("Watching for knative build pods in namespace %s\n", util.ColorInfo(ns))
	listWatch := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "pods", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		pod,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onPod(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onPod(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	// Wait forever
	select {}
}

func (o *ControllerBuildOptions) onPod(obj interface{}, jxClient versioned.Interface, ns string) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		log.Infof("Object is not a Pod %#v\n", obj)
		return
	}
	if pod != nil {
		labels := pod.Labels
		if labels != nil {
			buildName := labels[builds.LabelBuildName]
			if buildName != "" {
				log.Infof("Found build pod %s\n", pod.Name)

				activities := jxClient.JenkinsV1().PipelineActivities(ns)
				key := o.createPromoteStepActivityKey(buildName, pod)
				if key != nil {
					a, created, err := key.GetOrCreate(activities)
					if err != nil {
						operation := "update"
						if created {
							operation = "create"
						}
						log.Warnf("Failed to %s PipelineActivities for build %s: %s\n", operation, buildName, err)
					}

					if o.updatePipelineActivity(a, buildName, pod) {
						_, err := activities.Update(a)
						if err != nil {
							log.Warnf("Failed to update PipelineActivities%s: %s\n", a.Name, err)
						}
					}
				}
			}
		}
	}
}

// createPromoteStepActivityKey deduces the pipeline metadata from the knative build pod
func (o *ControllerBuildOptions) createPromoteStepActivityKey(buildName string, pod *corev1.Pod) *kube.PromoteStepActivityKey {
	branch := ""
	lastCommitSha := ""
	lastCommitMessage := ""
	lastCommitURL := ""
	build := digitSuffix(buildName)
	if build == "" {
		build = "1"
	}
	gitUrl := ""
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "build-step-git-source" {
			args := initContainer.Args
			for i := 0; i <= len(args)-2; i += 2 {
				key := args[i]
				value := args[i+1]

				switch key {
				case "-url":
					gitUrl = value
				case "-revision":
					branch = value
				}
			}
			break
		}
	}
	if gitUrl == "" {
		return nil
	}
	if branch == "" {
		branch = "master"
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		log.Warnf("Failed to parse git URL %s: %s", gitUrl, err)
		return nil
	}
	org := gitInfo.Organisation
	repo := gitInfo.Name
	name := org + "-" + repo + "-" + branch + "-" + build
	pipeline := org + "/" + repo + "/" + branch
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:              name,
			Pipeline:          pipeline,
			Build:             build,
			LastCommitSHA:     lastCommitSha,
			LastCommitMessage: lastCommitMessage,
			LastCommitURL:     lastCommitURL,
			GitInfo:           gitInfo,
		},
	}
}

func (o *ControllerBuildOptions) updatePipelineActivity(activity *v1.PipelineActivity, s string, pod *corev1.Pod) bool {
	copy := *activity
	// TODO update the steps based on the knative build pod's init containers
	for _, c := range pod.Status.InitContainerStatuses {
		name := strings.Replace(strings.TrimPrefix(c.Name, "build-step-"), "-", " ", -1)
		title := strings.Title(name)
		_, stage, _ := kube.GetOrCreateStage(activity, title)

		running := c.State.Running
		terminated := c.State.Terminated

		var startedAt metav1.Time
		var finishedAt metav1.Time
		if running != nil {
			startedAt = running.StartedAt
		} else if terminated != nil {
			startedAt = terminated.StartedAt
			finishedAt = terminated.FinishedAt
		}

		if !startedAt.IsZero() {
			stage.StartedTimestamp = &startedAt
		}
		if !finishedAt.IsZero() {
			stage.CompletedTimestamp = &finishedAt
		}
		stage.Description = createStepDescription(c.Name, pod)

		if terminated != nil {
			if terminated.ExitCode == 0 {
				stage.Status = v1.ActivityStatusTypeSucceeded
			} else {
				stage.Status = v1.ActivityStatusTypeFailed
			}
		} else {
			if running != nil {
				stage.Status = v1.ActivityStatusTypeRunning
			} else {
				stage.Status = v1.ActivityStatusTypePending
			}
		}
	}
	spec := &activity.Spec
	var biggestFinishedAt metav1.Time

	allCompleted := true
	failed := false
	running := true
	for _, step := range spec.Steps {
		stage := step.Stage
		if stage != nil {
			stageFinished := false
			if stage.StartedTimestamp != nil && spec.StartedTimestamp == nil {
				spec.StartedTimestamp = stage.StartedTimestamp
			}
			if stage.CompletedTimestamp != nil {
				t := stage.CompletedTimestamp
				if !t.IsZero() {
					stageFinished = true
					if biggestFinishedAt.IsZero() || t.After(biggestFinishedAt.Time) {
						biggestFinishedAt = *t
					}
				}
			}
			if stageFinished {
				if stage.Status != v1.ActivityStatusTypeSucceeded {
					failed = true
				}
			} else {
				allCompleted = false
			}
			if stage.Status == v1.ActivityStatusTypeRunning {
				running = true
			}
		}
	}
	if allCompleted {
		if failed {
			spec.Status = v1.ActivityStatusTypeFailed
		} else {
			spec.Status = v1.ActivityStatusTypeSucceeded
		}
		if !biggestFinishedAt.IsZero() {
			spec.CompletedTimestamp = &biggestFinishedAt
		}
	} else {
		if running {
			spec.Status = v1.ActivityStatusTypeRunning
		} else {
			spec.Status = v1.ActivityStatusTypePending
		}
	}
	return !reflect.DeepEqual(&copy, activity)
}

// createStepDescription uses the spec of the init container to return a description
func createStepDescription(initContainerName string, pod *corev1.Pod) string {
	for _, c := range pod.Spec.InitContainers {
		if c.Name == initContainerName {
			return strings.Join(c.Args, " ")
		}
	}
	return ""
}

func digitSuffix(text string) string {
	answer := ""
	for {
		l := len(text)
		if l == 0 {
			return answer
		}
		lastChar := text[l-1:]
		for _, rune := range lastChar {
			if !unicode.IsDigit(rune) {
				return answer
			} else {
				break
			}
		}
		answer = lastChar + answer
		text = text[0 : l-1]
	}
}
