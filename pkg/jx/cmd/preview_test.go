package cmd_test

import (
	"reflect"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/auth"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	gits_matchers "github.com/jenkins-x/jx/pkg/gits/mocks/matchers"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmd_mocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	cmd_matchers "github.com/jenkins-x/jx/pkg/jx/cmd/mocks/matchers"
	"github.com/jenkins-x/jx/pkg/kube"
	k8s_v1 "k8s.io/api/core/v1"
	k8s_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8s_cs_fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"

	cs_fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"

	"os"
	"testing"

	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_mocks "k8s.io/client-go/kubernetes/fake"
)

const (
	application    = "test-app"
	name           = "test-app-name"
	namespace      = "jx"
	gitHubLink     = "https://github.com/an-org/a-repo"
	gitHubUsername = "test-user-1"
	prNum          = 1
	prAuthor       = "the-pr-author"
	prOwner        = "the-pr-owner"
)

//Pegomock 'any' matcher for *string.
//(since these don't seem to get generated).
func anyPtrToString() *string {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf((*(*string))(nil)).Elem()))
	var nullValue *string
	return nullValue
}

//Pegomock 'any' matcher for *int.
//(since these don't seem to get generated).
func anyPtrToInt() *int {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf((*(*int))(nil)).Elem()))
	var nullValue *int
	return nullValue
}

func TestRun_CreateNewPreviewEnv(t *testing.T) {

	//TODO: t.Parallel()

	RegisterMockTestingT(t)

	//Environment variables expected to be found by jx preview when run in cluster:
	os.Setenv("GITHUB_USERNAME", gitHubUsername)
	os.Setenv("GITHUB_BEARER_TOKEN", "abc123def")
	os.Setenv(cmd.JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST, "MyOrganisation")
	os.Setenv(cmd.JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT, "5000")
	os.Setenv("JOB_NAME", "job")
	os.Setenv("BUILD_NUMBER", "1")
	os.Setenv(cmd.ORG, "MyOrganisation")
	//TODO: remove?
	os.Setenv(cmd.APP_NAME, "MyApp")
	os.Setenv(cmd.PREVIEW_VERSION, "v0.1.2")

	factory := cmd_mocks.NewMockFactory()

	previewOpts := &cmd.PreviewOptions{
		PromoteOptions: cmd.PromoteOptions{
			CommonOptions: cmd.CommonOptions{
				Factory:   factory,
				Out:       os.Stdout,
				In:        os.Stdin,
				BatchMode: true,
			},
			Application: application,
		},
		Namespace:    namespace,
		DevNamespace: "jx",
		Name:         name,
		SourceURL:    gitHubLink + ".git",
		PullRequest:  string(prNum),
	}

	nsObj := &k8s_v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jx-testing",
		},
	}

	secret := &k8s_v1.Secret{}
	mockKubeClient := kube_mocks.NewSimpleClientset(nsObj, secret)

	ingressConfig := &k8s_v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ConfigMapIngressConfig,
		},
		Data: map[string]string{"key1": "value1", "domain": "test-domain", "config.yml": ""},
	}
	mockKubeClient.CoreV1().ConfigMaps(namespace).Create(ingressConfig)

	service := &k8s_v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-app",
			Annotations: map[string]string{kube.ExposeURLAnnotation: "http://the-service-url/with/a/path"},
		},
	}
	mockKubeClient.CoreV1().Services("jx").Create(service)

	var apiClient k8s_cs.Interface = &k8s_cs_fake.Clientset{}
	// Override CreateClient to return mock Kubernetes interface
	When(factory.CreateClient()).ThenReturn(mockKubeClient, "jx-testing", nil)
	When(factory.CreateApiExtensionsClient()).ThenReturn(apiClient, nil)

	//Setup Git mocks:
	mockGitProvider := gits_test.NewMockGitProvider()
	When(factory.CreateGitProvider(AnyString(), //gitURL
		AnyString(), //message
		cmd_matchers.AnyAuthAuthConfigService(),
		AnyString(), //gitKind
		AnyBool(),   //batchMode,
		cmd_matchers.AnyGitsGitter(),
		cmd_matchers.AnyTerminalFileReader(),
		cmd_matchers.AnyTerminalFileWriter(),
		cmd_matchers.AnyIoWriter(),
	)).ThenReturn(mockGitProvider, nil)
	number := prNum
	//todo: fill in e-mail & validate UserDetailService updates (when set).
	mockGitPR := &gits.GitPullRequest{
		Owner:  prOwner,
		Author: &gits.GitUser{Name: prAuthor},
		Number: &number,
	}
	When(mockGitProvider.GetPullRequest(AnyString(), //owner
		gits_matchers.AnyPtrToGitsGitRepositoryInfo(), //repo
		AnyInt(), // number
	)).ThenReturn(mockGitPR, nil)

	mockAuthConfigService := auth.AuthConfigService{}
	When(factory.CreateAuthConfigService(cmd.GitAuthConfigFile)).ThenReturn(mockAuthConfigService, nil)
	When(factory.IsInCDPIpeline()).ThenReturn(true)

	//env := &jio_v1.Environment{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: "my-app-name",
	//	},
	//}

	//cs := cs_fake.NewSimpleClientset(env)
	cs := cs_fake.NewSimpleClientset()
	When(factory.CreateJXClient()).ThenReturn(cs, namespace, nil)

	mockHelmer := helm_test.NewMockHelmer()
	When(factory.GetHelm(AnyBool(), AnyString(), AnyBool(), AnyBool())).ThenReturn(mockHelmer)
	When(mockHelmer.UpgradeChart(AnyString(),
		AnyString(),
		AnyString(),
		anyPtrToString(),
		AnyBool(),
		anyPtrToInt(),
		AnyBool(),
		AnyBool(),
		AnyStringSlice(),
		AnyStringSlice())).ThenReturn(nil) //err=nil

	err := previewOpts.Run()

	assert.NoError(t, err, "Should not error")

	envs := cs.JenkinsV1().Environments(namespace)
	previewEnv, err := envs.Get(name, metav1.GetOptions{})
	assert.NoError(t, err, "Preview environment should have been created.")
	assert.Equal(t, namespace, previewEnv.Namespace)
	assert.Equal(t, name, previewEnv.Name)
	assert.NotNil(t, previewEnv.Spec)
	assert.Equal(t, v1.EnvironmentKindTypePreview, previewEnv.Spec.Kind)
	assert.Equal(t, v1.PromotionStrategyTypeAutomatic, previewEnv.Spec.PromotionStrategy)
	prURL := gitHubLink + "/pull/" + string(prNum)
	assert.Equal(t, prURL, previewEnv.Spec.PullRequestURL)
	//todo: validate previewgitspec
	assert.NotNil(t, previewEnv.Spec.PreviewGitSpec)
	assert.Equal(t, prNum, previewEnv.Spec.PreviewGitSpec.Name)
	assert.Equal(t, prURL, previewEnv.Spec.PreviewGitSpec.URL)
	//todo: set build status.
	assert.Equal(t, "", previewEnv.Spec.PreviewGitSpec.BuildStatus)
	assert.Equal(t, "", previewEnv.Spec.PreviewGitSpec.BuildStatusURL)
	assert.Equal(t, gitHubUsername, previewEnv.Spec.PreviewGitSpec.User)

	//TODO: assert CRD registrations?

	//TODO: check PR comment.

}
