package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd/api"

	cmd_mocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	kuber_mocks "github.com/jenkins-x/jx/pkg/kube/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"testing"

	. "github.com/petergtz/pegomock"
)

func setupUninstall(contextName string) (*cmd_mocks.MockFactory, *kuber_mocks.MockKuber) {
	factory := cmd_mocks.NewMockFactory()
	kubeMock := kuber_mocks.NewMockKuber()
	fakeKubeConfig := api.NewConfig()
	fakeKubeConfig.CurrentContext = contextName
	When(kubeMock.LoadConfig()).ThenReturn(fakeKubeConfig, nil, nil)
	return factory, kubeMock
}

func TestUninstallOptions_Run_ContextSpecifiedAsOption_FailsWhenContextNamesDoNotMatch(t *testing.T) {
	factory, kubeMock := setupUninstall("current-context")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Factory: factory,
			Kuber:   kubeMock,
		},
		Namespace: "ns",
		Context:   "target-context",
	}

	err := o.Run()
	assert.EqualError(t, err, "The context 'target-context' must match the current context to uninstall")
}

func TestUninstallOptions_Run_ContextSpecifiedAsOption_PassWhenContextNamesMatch(t *testing.T) {
	factory, kubeMock := setupUninstall("correct-context-to-delete")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Factory: factory,
			Kuber:   kubeMock,
		},
		Namespace: "ns",
		Context:   "correct-context-to-delete",
	}
	cmd.ConfigureTestOptions(&o.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())

	// Create fake namespace (that we will uninstall from)
	err := createNamespace(o, "ns")

	// Run the uninstall
	err = o.Run()
	assert.NoError(t, err)

	// Assert that the namespace has been deleted
	_, err = o.KubeClientCached.CoreV1().Namespaces().Get("ns", metav1.GetOptions{})
	assert.Error(t, err)
}

func TestUninstallOptions_Run_ContextSpecifiedAsOption_PassWhenForced(t *testing.T) {
	factory, kubeMock := setupUninstall("correct-context-to-delete")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Factory: factory,
			Kuber:   kubeMock,
		},
		Namespace: "ns",
		Force:     true,
	}
	cmd.ConfigureTestOptions(&o.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())

	// Create fake namespace (that we will uninstall from)
	err := createNamespace(o, "ns")

	// Run the uninstall
	err = o.Run()
	assert.NoError(t, err)

	// Assert that the namespace has been deleted
	_, err = o.KubeClientCached.CoreV1().Namespaces().Get("ns", metav1.GetOptions{})
	assert.Error(t, err)
}

func createNamespace(o *cmd.UninstallOptions, ns string) error {
	_, err := o.KubeClientCached.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	return err
}

// TODO: Interaction-based tests with the CLI
