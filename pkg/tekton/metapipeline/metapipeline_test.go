package metapipeline

import (
	"github.com/jenkins-x/jx/pkg/prow"
	"path/filepath"
	"testing"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
)

func TestMetaPipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meta pipeline Suite")
}

var _ = Describe("Meta pipeline", func() {

	Describe("#CreateMetaPipelineCRDs", func() {
		var (
			testParams   CRDCreationParameters
			actualCRDs   *tekton.CRDWrapper
			actualError  error
			actualStdout string
		)

		BeforeEach(func() {
			gitInfo, _ := gits.NewGitFake().Info("/acme")
			pullRef, _ := prow.ParsePullRefs("master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,42:db8e2d275df53477b1c6871f7d7f4281dacf3169")

			testParams = CRDCreationParameters{
				PipelineName:     "test-pipeline",
				PipelineKind:     "pullrequest",
				BranchIdentifier: "master",
				PullRef:          *pullRef,
				GitInfo:          *gitInfo,
				Labels:           []string{"someLabel=someValue"},
				EnvVars:          []string{"SOME_VAR=SOME_VAL"},
				BuildNumber:      "1",
				SourceDir:        "source",
				ServiceAccount:   "tekton-bot",
				VersionsDir:      filepath.Join("test_data", "stable_versions"),
			}
		})

		JustBeforeEach(func() {
			actualCRDs, actualStdout, actualError = createMetaPipeline(testParams)
		})

		Context("with no extending Apps", func() {
			It("should not error", func() {
				Expect(actualError).Should(BeNil())
			})

			It("should not write to stdout", func() {
				Expect(actualStdout).Should(BeEmpty())
			})

			It("contain a single task", func() {
				tasks := actualCRDs.Tasks()
				Expect(tasks).Should(HaveLen(1))
			})

			It("contain four task steps", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				Expect(steps).Should(HaveLen(4))
				Expect(steps[0].Name).Should(Equal("git-merge"))
				Expect(steps[1].Name).Should(Equal(mergePullRefsStepName))
				Expect(steps[2].Name).Should(Equal(createEffectivePipelineStepName))
				Expect(steps[3].Name).Should(Equal(createTektonCRDsStepName))
			})

			It("merge pull refs step passes correct pull ref", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				mergePullRefStep := steps[1]
				Expect(mergePullRefStep.Args).Should(Equal([]string{"jx step git merge --baseBranch master --baseSHA 0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815 --sha db8e2d275df53477b1c6871f7d7f4281dacf3169"}))
			})

			It("should have correct step create task args", func() {
				step := actualCRDs.Tasks()[0].Spec.Steps[3]
				Expect(step.Args).Should(Equal([]string{"jx step create task --clone-dir /workspace/source --kind pullrequest --pr-number 42 --service-account tekton-bot --source source --branch master --build-number 1 --label someLabel=someValue --env SOME_VAR=SOME_VAL"}))
			})
		})

		Context("with extending App missing required metadata", func() {
			JustBeforeEach(func() {
				testApp := jenkinsv1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "acme-app",
						Labels: map[string]string{apps.AppTypeLabel: apps.PipelineExtension.String()},
					},

					Spec: jenkinsv1.AppSpec{},
				}
				testParams.Apps = []jenkinsv1.App{testApp}
				actualCRDs, actualStdout, actualError = createMetaPipeline(testParams)
			})

			It("should not error", func() {
				Expect(actualError).Should(BeNil())
			})

			It("should write warning to stdout", func() {
				Expect(actualStdout).Should(ContainSubstring("WARNING: Skipping app acme-app in meta pipeline"))
			})

			It("contain a single task", func() {
				tasks := actualCRDs.Tasks()
				Expect(tasks).Should(HaveLen(1))
			})

			It("contain three task steps", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				Expect(steps).Should(HaveLen(4))
				Expect(steps[0].Name).Should(Equal("git-merge"))
				Expect(steps[1].Name).Should(Equal(mergePullRefsStepName))
				Expect(steps[2].Name).Should(Equal(createEffectivePipelineStepName))
				Expect(steps[3].Name).Should(Equal(createTektonCRDsStepName))
			})
		})

		Context("with extending App", func() {
			JustBeforeEach(func() {
				testApp := jenkinsv1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "acme-app",
						Labels: map[string]string{apps.AppTypeLabel: apps.PipelineExtension.String()},
					},

					Spec: jenkinsv1.AppSpec{
						PipelineExtension: &jenkinsv1.PipelineExtension{
							Name:    "acme-ext",
							Image:   "acme:1.0.0",
							Command: "run",
							Args:    []string{},
						},
					},
				}
				testParams.Apps = []jenkinsv1.App{testApp}
				actualCRDs, actualStdout, actualError = createMetaPipeline(testParams)
			})

			It("should not error", func() {
				Expect(actualError).Should(BeNil())
			})

			It("should not write to stdout", func() {
				Expect(actualStdout).Should(BeEmpty())
			})

			It("contain a single task", func() {
				tasks := actualCRDs.Tasks()
				Expect(tasks).Should(HaveLen(1))
			})

			It("contain three task steps", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				Expect(steps).Should(HaveLen(5))
				Expect(steps[0].Name).Should(Equal("git-merge"))
				Expect(steps[1].Name).Should(Equal(mergePullRefsStepName))
				Expect(steps[2].Name).Should(Equal(createEffectivePipelineStepName))
				Expect(steps[3].Name).Should(Equal("acme-ext"))
				Expect(steps[4].Name).Should(Equal(createTektonCRDsStepName))
			})
		})

		Context("with no SHAs to merge (only baseBranch)", func() {
			JustBeforeEach(func() {
				pullRef, err := prow.ParsePullRefs("master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815")
				Expect(err).Should(BeNil())
				testParams.PullRef = *pullRef
				actualCRDs, actualStdout, actualError = createMetaPipeline(testParams)
			})

			It("merge pull refs step passes correct pull ref", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				mergePullRefStep := steps[1]
				Expect(mergePullRefStep.Args[0]).Should(ContainSubstring("SKIP merge-pull-refs: Nothing to merge"))
			})
		})

		Context("with no SHAs to merge (baseBranch & baseSHA)", func() {
			JustBeforeEach(func() {
				pullRef, err := prow.ParsePullRefs("master:")
				Expect(err).Should(BeNil())
				testParams.PullRef = *pullRef
				actualCRDs, actualStdout, actualError = createMetaPipeline(testParams)
			})

			It("merge pull refs step passes correct pull ref", func() {
				steps := actualCRDs.Tasks()[0].Spec.Steps
				mergePullRefStep := steps[1]
				Expect(mergePullRefStep.Args[0]).Should(ContainSubstring("SKIP merge-pull-refs: Nothing to merge"))
			})
		})
	})

	Describe("#GetExtendingApps", func() {
		var (
			jxClient versioned.Interface
		)

		BeforeEach(func() {
			jxClient = fake.NewSimpleClientset()
		})

		Context("with no extending App", func() {
			It("should return empty app list", func() {
				apps, err := GetExtendingApps(jxClient, "jx")
				Expect(err).Should(BeNil())
				Expect(apps).Should(HaveLen(0))
			})
		})

		Context("with extending App", func() {
			BeforeEach(func() {
				testApp := jenkinsv1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "acme-app",
						Labels: map[string]string{apps.AppTypeLabel: apps.PipelineExtension.String()},
					},

					Spec: jenkinsv1.AppSpec{
						PipelineExtension: &jenkinsv1.PipelineExtension{
							Name:    "acme-ext",
							Image:   "acme:1.0.0",
							Command: "run",
							Args:    []string{},
						},
					},
				}
				_, err := jxClient.JenkinsV1().Apps("jx").Create(&testApp)
				Expect(err).Should(BeNil())
			})

			It("should return the registered App", func() {
				apps, err := GetExtendingApps(jxClient, "jx")
				Expect(err).Should(BeNil())
				Expect(apps).Should(HaveLen(1))
				Expect(apps[0].Name).Should(Equal("acme-app"))
			})
		})
	})
})

func createMetaPipeline(params CRDCreationParameters) (*tekton.CRDWrapper, string, error) {
	var crds *tekton.CRDWrapper
	var err error
	out := log.CaptureOutput(func() {
		crds, err = CreateMetaPipelineCRDs(params)
	})
	return crds, out, err
}
