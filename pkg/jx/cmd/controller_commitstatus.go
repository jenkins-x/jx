package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/extensions"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/builds"

	corev1 "k8s.io/api/core/v1"

	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/log"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// ControllerCommitStatusOptions the options for the controller
type ControllerCommitStatusOptions struct {
	ControllerOptions
}

// NewCmdControllerCommitStatus creates a command object for the "create" command
func NewCmdControllerCommitStatus(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerCommitStatusOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "commitstatus",
		Short: "Updates commit status",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "v", false, "Enable verbose logging")
	return cmd
}

// Run implements this command
func (o *ControllerCommitStatusOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterCommitStatusCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	commitstatusListWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "commitstatuses", ns, fields.Everything())
	kube.SortListWatchByName(commitstatusListWatch)
	_, commitstatusController := cache.NewInformer(
		commitstatusListWatch,
		&jenkinsv1.CommitStatus{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onCommitStatusObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onCommitStatusObj(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {

			},
		},
	)
	stop := make(chan struct{})
	go commitstatusController.Run(stop)

	podListWatch := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", ns, fields.Everything())
	kube.SortListWatchByName(podListWatch)
	_, podWatch := cache.NewInformer(
		podListWatch,
		&corev1.Pod{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onPodObj(obj, jxClient, kubeClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onPodObj(newObj, jxClient, kubeClient, ns)
			},
			DeleteFunc: func(obj interface{}) {

			},
		},
	)
	stop = make(chan struct{})
	podWatch.Run(stop)

	if err != nil {
		return err
	}
	return nil
}

func (o *ControllerCommitStatusOptions) onCommitStatusObj(obj interface{}, jxClient jenkinsv1client.Interface, ns string) {
	check, ok := obj.(*jenkinsv1.CommitStatus)
	if !ok {
		log.Fatalf("commit status controller: unexpected type %v\n", obj)
	} else {
		err := o.onCommitStatus(check, jxClient, ns)
		if err != nil {
			log.Fatalf("commit status controller: %v\n", err)
		}
	}
}

func (o *ControllerCommitStatusOptions) onCommitStatus(check *jenkinsv1.CommitStatus, jxClient jenkinsv1client.Interface, ns string) error {
	err := o.Check(check, jxClient, ns)
	if err != nil {
		gitProvider, gitRepoInfo, err1 := o.createGitProviderForURLWithoutKind(check.Spec.Commit.GitURL)
		if err1 != nil {
			return err1
		}
		_, err1 = extensions.NotifyCommitStatus(check.Spec.Commit, "error", "", "Internal Error performing commit status updates", "", check.Spec.Context, gitProvider, gitRepoInfo)
		if err1 != nil {
			return err
		}
		return err
	}
	return nil
}

func (o *ControllerCommitStatusOptions) onPodObj(obj interface{}, jxClient jenkinsv1client.Interface, kubeClient kubernetes.Interface, ns string) {
	check, ok := obj.(*corev1.Pod)
	if !ok {
		log.Fatalf("pod watcher: unexpected type %v\n", obj)
	} else {
		err := o.onPod(check, jxClient, kubeClient, ns)
		if err != nil {
			log.Fatalf("pod watcher: %v\n", err)
		}
	}
}

func (o *ControllerCommitStatusOptions) onPod(pod *corev1.Pod, jxClient jenkinsv1client.Interface, kubeClient kubernetes.Interface, ns string) error {
	if pod != nil {
		labels := pod.Labels
		if labels != nil {
			buildName := labels[builds.LabelBuildName]
			if buildName == "" {
				buildName = labels[builds.LabelOldBuildName]
			}
			if buildName != "" {
				org := ""
				repo := ""
				pullRequest := ""
				pullPullSha := ""
				pullBaseSha := ""
				buildNumber := ""
				sourceUrl := ""
				branch := ""
				for _, initContainer := range pod.Spec.InitContainers {
					for _, e := range initContainer.Env {
						switch e.Name {
						case "REPO_OWNER":
							org = e.Value
						case "REPO_NAME":
							repo = e.Value
						case "PULL_NUMBER":
							pullRequest = fmt.Sprintf("PR-%s", e.Value)
						case "PULL_PULL_SHA":
							pullPullSha = e.Value
						case "PULL_BASE_SHA":
							pullBaseSha = e.Value
						case "JX_BUILD_NUMBER":
							buildNumber = e.Value
						case "SOURCE_URL":
							sourceUrl = e.Value
						case "PULL_BASE_REF":
							branch = e.Value
						}
					}
				}
				if org != "" && repo != "" && buildNumber != "" && (pullBaseSha != "" || pullPullSha != "") {

					sha := pullBaseSha
					if pullRequest != "PR-" {
						sha = pullPullSha
						branch = pullRequest
					}
					if o.Verbose {
						log.Infof("pod watcher: build pod: %s, org: %s, repo: %s, buildNumber: %s, pullBaseSha: %s, pullPullSha: %s, pullRequest: %s, sourceUrl: %s\n", pod.Name, org, repo, buildNumber, pullBaseSha, pullPullSha, pullRequest, sourceUrl)
					}
					if sha == "" {
						log.Warnf("No sha on %s, not upserting commit status\n", pod.Name)
					} else {
						cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get("jenkins-x-extensions", metav1.GetOptions{})
						if err != nil {
							return err
						}
						commitstatusContextsYaml, ok := cm.Data["commitstatusContexts"]
						if ok {
							commitStatusContexts := make([]string, 0)
							err = yaml.Unmarshal([]byte(commitstatusContextsYaml), &commitStatusContexts)
							if err != nil {
								return err
							}
							for _, ctx := range commitStatusContexts {
								name := kube.ToValidName(fmt.Sprintf("%s-%s-%s-%s-%s", org, repo, branch, buildNumber, ctx))
								err = o.UpsertCommitStatusCheck(name, sourceUrl, sha, pullRequest, ctx, jxClient, ns)
								if err != nil {
									return err
								}
							}
						} else {
							log.Infof("No contexts defined to upsert commit status for %s\n", pod.Name)
							return nil
						}

					}
				}
			}

		}

	}
	return nil
}

func (o *ControllerCommitStatusOptions) UpsertCommitStatusCheck(name string, url string, sha string, pullRequest string, context string, jxClient jenkinsv1client.Interface, ns string) error {
	if name != "" {

		check, err := jxClient.JenkinsV1().CommitStatuses(ns).Get(name, metav1.GetOptions{})
		create := false
		update := false
		actRef := jenkinsv1.ResourceReference{}
		if err != nil {
			create = true
		} else {
			log.Infof("commit status controller: commit status already exists for %s\n", name)
		}
		if create || check.Spec.PipelineActivity.UID == "" {
			act, err := jxClient.JenkinsV1().PipelineActivities(ns).Get(name, metav1.GetOptions{})
			if err == nil {
				update = true
				actRef.Name = act.Name
				actRef.Kind = act.Kind
				actRef.UID = act.UID
				actRef.APIVersion = act.APIVersion
			}
		}

		if create {
			log.Infof("commit status controller: Creating commit status for %s\n", name)
			_, err := jxClient.JenkinsV1().CommitStatuses(ns).Create(&jenkinsv1.CommitStatus{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"lastCommitSha": sha,
					},
				},
				Spec: jenkinsv1.CommitStatusSpec{
					Checked: false,
					Commit: jenkinsv1.CommitStatusCommitReference{
						GitURL:      url,
						PullRequest: pullRequest,
						SHA:         sha,
					},
					PipelineActivity: actRef,
					Context:          context,
				},
			})
			if err != nil {
				return err
			}
		} else if update {
			check.Spec.PipelineActivity = actRef
			_, err := jxClient.JenkinsV1().CommitStatuses(ns).Update(check)
			if err != nil {
				return err
			}
		}

	} else {
		errors.New("commit status controller: Must supply name")
	}
	return nil
}

func (o *ControllerCommitStatusOptions) Check(check *jenkinsv1.CommitStatus, jxClient jenkinsv1client.Interface, ns string) error {
	gitProvider, gitRepoInfo, err := o.createGitProviderForURLWithoutKind(check.Spec.Commit.GitURL)
	if err != nil {
		return err
	}
	pass := false
	if check.Spec.Checked {
		var commentBuilder strings.Builder
		pass = true
		for _, c := range check.Spec.Items {
			if !c.Pass {
				pass = false
				fmt.Fprintf(&commentBuilder, "%s | %s | %s | TODO | `/test this`\n", c.Name, c.Description, check.Spec.Commit.SHA)
			}
		}
		if pass {
			_, err := extensions.NotifyCommitStatus(check.Spec.Commit, "success", "", "%s completed successfully", "", check.Spec.Context, gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		} else {
			comment := fmt.Sprintf(
				"The following commit status checks **failed**, say `/retest` to rerun them all:\n"+
					"\n"+
					"Name | Description | Commit | Details | Rerun command\n"+
					"--- | --- | --- | --- | --- \n"+
					"%s\n"+
					"<details>\n"+
					"\n"+
					"Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).\n"+
					"</details>", commentBuilder.String())
			_, err := extensions.NotifyCommitStatus(check.Spec.Commit, "failure", "", "Some commit status checks failed", comment, check.Spec.Context, gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		}
	} else {
		latest := true
		// TODO This could be improved with labels
		checks, err := jxClient.JenkinsV1().CommitStatuses(ns).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, c := range checks.Items {
			if c.Spec.Commit.GitURL == check.Spec.Commit.GitURL && c.Spec.Commit.PullRequest == check.Spec.Commit.PullRequest && c.CreationTimestamp.UnixNano() > check.CreationTimestamp.UnixNano() {
				latest = false
				break
			}
		}
		if latest {
			_, err = extensions.NotifyCommitStatus(check.Spec.Commit, "pending", "", "Waiting for commit status checks to complete", "", check.Spec.Context, gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
