package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/builds"

	corev1 "k8s.io/api/core/v1"

	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/governance"

	"github.com/jenkins-x/jx/pkg/log"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// ControllerComplianceOptions the options for the controller compliance
type ControllerComplianceOptions struct {
	ControllerOptions
}

// NewCmdControllerCompliance creates a command object for the "create" command
func NewCmdControllerCompliance(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerComplianceOptions{
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
		Use:   "compliance",
		Short: "Enforces compliance",
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
func (o *ControllerComplianceOptions) Run() error {
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
	err = kube.RegisterComplianceCheckCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	complianceListWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "compliancechecks", ns, fields.Everything())
	kube.SortListWatchByName(complianceListWatch)
	_, complianceController := cache.NewInformer(
		complianceListWatch,
		&jenkinsv1.ComplianceCheck{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onComplianceCheckObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onComplianceCheckObj(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {

			},
		},
	)
	stop := make(chan struct{})
	go complianceController.Run(stop)

	podListWatch := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", ns, fields.Everything())
	kube.SortListWatchByName(podListWatch)
	_, podWatch := cache.NewInformer(
		podListWatch,
		&corev1.Pod{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onPodObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onPodObj(newObj, jxClient, ns)
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

func (o *ControllerComplianceOptions) onComplianceCheckObj(obj interface{}, jxClient jenkinsv1client.Interface, ns string) {
	check, ok := obj.(*jenkinsv1.ComplianceCheck)
	if !ok {
		log.Fatalf("compliance controller: unexpected type %v\n", obj)
	} else {
		err := o.onComplianceCheck(check, jxClient, ns)
		if err != nil {
			log.Fatalf("compliance controller: %v\n", err)
		}
	}
}

func (o *ControllerComplianceOptions) onComplianceCheck(check *jenkinsv1.ComplianceCheck, jxClient jenkinsv1client.Interface, ns string) error {
	err := o.Check(check.Spec)
	if err != nil {
		gitProvider, gitRepoInfo, err1 := o.createGitProviderForURLWithoutKind(check.Spec.Commit.GitURL)
		if err1 != nil {
			return err1
		}
		_, err1 = governance.NotifyComplianceState(check.Spec.Commit, "error", "", "Internal Error performing compliance checks", "", gitProvider, gitRepoInfo)
		if err1 != nil {
			return err
		}
		return err
	}
	return nil
}

func (o *ControllerComplianceOptions) onPodObj(obj interface{}, jxClient jenkinsv1client.Interface, ns string) {
	check, ok := obj.(*corev1.Pod)
	if !ok {
		log.Fatalf("pod watcher: unexpected type %v\n", obj)
	} else {
		err := o.onPod(check, jxClient, ns)
		if err != nil {
			log.Fatalf("pod watcher: %v\n", err)
		}
	}
}

func (o *ControllerComplianceOptions) onPod(pod *corev1.Pod, jxClient jenkinsv1client.Interface, ns string) error {
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
					}
					if o.Verbose {
						log.Infof("pod watcher: build pod: %s, org: %s, repo: %s, buildNumber: %s, pullBaseSha: %s, pullPullSha: %s, pullRequest: %s, sourceUrl: %s\n", pod.Name, org, repo, buildNumber, pullBaseSha, pullPullSha, pullRequest, sourceUrl)
					}
					if sha == "" {
						log.Warnf("No sha on %s, not upserting compliance check\n", pod.Name)
					} else {
						name := kube.ToValidName(fmt.Sprintf("%s-%s-%s-%s", org, repo, branch, buildNumber))
						return o.UpsertComplianceCheck(name, sourceUrl, sha, pullRequest, jxClient, ns)
					}
				}
			}

		}

	}
	return nil
}

func (o *ControllerComplianceOptions) UpsertComplianceCheck(name string, url string, sha string, pullRequest string, jxClient jenkinsv1client.Interface, ns string) error {
	if name != "" {

		check, err := jxClient.JenkinsV1().ComplianceChecks(ns).Get(name, metav1.GetOptions{})
		create := false
		update := false
		actRef := jenkinsv1.ResourceReference{}
		if err != nil {
			create = true
		} else {
			log.Infof("compliance controller: Compliance Check already exists for %s\n", name)
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
			log.Infof("compliance controller: Creating compliance check for %s\n", name)
			_, err := jxClient.JenkinsV1().ComplianceChecks(ns).Create(&jenkinsv1.ComplianceCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"lastCommitSha": sha,
					},
				},
				Spec: jenkinsv1.ComplianceCheckSpec{
					Checked: false,
					Commit: jenkinsv1.ComplianceCheckCommitReference{
						GitURL:      url,
						PullRequest: pullRequest,
						SHA:         sha,
					},
					PipelineActivity: actRef,
				},
			})
			if err != nil {
				return err
			}
		} else if update {
			check.Spec.PipelineActivity = actRef
			_, err := jxClient.JenkinsV1().ComplianceChecks(ns).Update(check)
			if err != nil {
				return err
			}
		}

	} else {
		errors.New("compliance controller: Must supply name")
	}
	return nil
}

func (o *ControllerComplianceOptions) Check(check jenkinsv1.ComplianceCheckSpec) error {
	gitProvider, gitRepoInfo, err := o.createGitProviderForURLWithoutKind(check.Commit.GitURL)
	if err != nil {
		return err
	}
	pass := false
	if check.Checked {
		var commentBuilder strings.Builder
		pass = true
		for _, c := range check.Checks {
			if !c.Pass {
				pass = false
				fmt.Fprintf(&commentBuilder, "%s | %s | %s | TODO | `/test this`\n", c.Name, c.Description, check.Commit.SHA)
			}
		}
		if pass {
			_, err := governance.NotifyComplianceState(check.Commit, "success", "", "Compliance checks completed successfully", "", gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		} else {

			comment := fmt.Sprintf(
				"The following compliance checks **failed**, say `/retest` to rerun them all:\n"+
					"\n"+
					"Name | Description | Commit | Details | Rerun command\n"+
					"--- | --- | --- | --- | --- \n"+
					"%s\n"+
					"<details>\n"+
					"\n"+
					"Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).\n"+
					"</details>", commentBuilder.String())
			_, err := governance.NotifyComplianceState(check.Commit, "failure", "", "Some compliance checks failed", comment, gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		}
	} else {
		_, err := governance.NotifyComplianceState(check.Commit, "pending", "", "Waiting for compliance checks to complete", "", gitProvider, gitRepoInfo)
		if err != nil {
			return err
		}
	}
	return nil
}
