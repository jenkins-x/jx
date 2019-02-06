package cmd_test

import (
	"testing"

	expect "github.com/Netflix/go-expect"
	gojenkins "github.com/jenkins-x/golang-jenkins"
	jenkins_mocks "github.com/jenkins-x/golang-jenkins/mocks"
	versiond_mocks "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/config"
	gits_mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmd_mocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	cmd_mock_matchers "github.com/jenkins-x/jx/pkg/jx/cmd/mocks/matchers"
	"github.com/jenkins-x/jx/pkg/tests"
	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"k8s.io/api/core/v1"
	apiextentions_mocks "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_mocks "k8s.io/client-go/kubernetes/fake"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestCreateEnvRun(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	// namespace fixture
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jx-testing",
		},
	}

	exposeControllerConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exposecontroller",
			Namespace: "jx-testing",
		},
	}

	jenkinsConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "jx-testing",
		},
	}

	// mock gitProvider
	gitProviderInterface := gits_mocks.NewMockGitProvider()

	// mock factory
	factory := cmd_mocks.NewMockFactory()
	When(factory.CreateAuthConfigService(AnyString())).ThenReturn(tests.CreateAuthConfigService(), nil)
	When(factory.CreateGitProvider(
		AnyString(),
		AnyString(),
		cmd_mock_matchers.AnyAuthConfigService(),
		AnyString(),
		AnyBool(),
		cmd_mock_matchers.AnyGitsGitter(),
		cmd_mock_matchers.AnyTerminalFileReader(),
		cmd_mock_matchers.AnyTerminalFileWriter(),
		cmd_mock_matchers.AnyIoWriter(),
	)).ThenReturn(gitProviderInterface, nil)

	// mock Kubernetes interface
	kubernetesInterface := kube_mocks.NewSimpleClientset(namespace, exposeControllerConfigMap, jenkinsConfigMap)
	// Override CreateKubeClient to return mock Kubernetes interface
	When(factory.CreateKubeClient()).ThenReturn(kubernetesInterface, "jx-testing", nil)
	//When(kubernetesInterface.CoreV1().Namespaces().Update(namespace)).ThenReturn(namespace, nil)

	// mock versiond interface
	versiondInterface := versiond_mocks.NewSimpleClientset()
	// Override CreateJXClient to return mock versiond interface
	When(factory.CreateJXClient()).ThenReturn(versiondInterface, "jx-testing", nil)

	// mock apiExtensions interface
	apiextensionsInterface := apiextentions_mocks.NewSimpleClientset()
	// Override CreateApiExtensionsClient to return mock apiextensions interface
	When(factory.CreateApiExtensionsClient()).ThenReturn(apiextensionsInterface, nil)

	// mock Jenkins client
	jenkinsClientInterface := jenkins_mocks.NewMockJenkinsClient()
	When(factory.CreateJenkinsClient(cmd_mock_matchers.AnyKubernetesInterface(), AnyString(), cmd_mock_matchers.AnyTerminalFileReader(), cmd_mock_matchers.AnyTerminalFileWriter(), cmd_mock_matchers.AnyIoWriter())).ThenReturn(jenkinsClientInterface, nil)
	jenkinsJob := gojenkins.Job{Class: "com.cloudbees.hudson.plugins.folder.Folder"}
	When(jenkinsClientInterface.GetJob(AnyString())).ThenReturn(jenkinsJob, nil)

	// Mock terminal
	console := tests.NewTerminal(t)

	// Test interactive IO
	donec := make(chan struct{})
	//noinspection GoUnhandledErrorResult
	go func() {
		defer close(donec)
		console.ExpectString("Name:")
		console.SendLine("testing")
		console.ExpectString("Label:")
		console.SendLine("Testing")
		console.ExpectString("Namespace:")
		console.SendLine("jx-testing")
		console.ExpectString("Cluster URL:")
		console.SendLine("http://good.looking.com")
		console.ExpectString("Promotion Strategy:")
		console.SendLine("A")
		console.ExpectString("Order:")
		console.SendLine("1")
		console.ExpectString("We will now create a Git repository to store your testing environment, ok? :")
		console.SendLine("N")
		console.ExpectString("Git URL for the Environment source code:")
		console.SendLine("https://github.com/jx-testing-user/testing-env")
		console.ExpectString("Git branch for the Environment source code:")
		console.SendLine("master")
		console.ExpectString("Do you wish to use jx-testing-user as the user name for the Jenkins Pipeline")
		console.SendLine("Y")
		console.ExpectEOF()
	}()

	a := make(map[string]string)
	a["helm.sh/hook"] = "post-install,post-upgrade"
	a["helm.sh/hook-delete-policy"] = "hook-succeeded"
	helmValuesConfig := config.HelmValuesConfig{
		ExposeController: &config.ExposeController{},
	}
	helmValuesConfig.ExposeController.Annotations = a
	helmValuesConfig.ExposeController.Config.Exposer = "Ingress"
	helmValuesConfig.ExposeController.Config.Domain = "jx-testing.com"
	helmValuesConfig.ExposeController.Config.HTTP = "false"
	helmValuesConfig.ExposeController.Config.TLSAcme = "false"

	options := cmd.CreateEnvOptions{
		HelmValuesConfig: helmValuesConfig,
		CreateOptions: cmd.CreateOptions{
			CommonOptions: cmd.CommonOptions{
				Factory: factory,
				In:      console.In,
				Out:     console.Out,
				Err:     console.Err,
			},
		},
	}

	err := options.Run()

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	console.Close()
	<-donec

	assert.NoError(t, err, "Should not error")

	// Dump the terminal's screen.
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))
}
