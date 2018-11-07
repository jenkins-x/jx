package cmd_test

import (
	"github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	"github.com/jenkins-x/jx/pkg/tests"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd/api"

	kuber_mocks "github.com/jenkins-x/jx/pkg/kube/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"testing"

	. "github.com/petergtz/pegomock"
)

func setupUninstall(contextName string) *kuber_mocks.MockKuber {
	kubeMock := kuber_mocks.NewMockKuber()
	fakeKubeConfig := api.NewConfig()
	fakeKubeConfig.CurrentContext = contextName
	When(kubeMock.LoadConfig()).ThenReturn(fakeKubeConfig, nil, nil)
	return kubeMock
}

func TestUninstallOptions_Run_ContextSpecifiedAsOption_FailsWhenContextNamesDoNotMatch(t *testing.T) {
	kubeMock := setupUninstall("current-context")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			//Factory: factory,
			Kuber: kubeMock,
		},
		Namespace: "ns",
		Context:   "target-context",
	}
	cmd.ConfigureTestOptions(&o.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())

	err := o.Run()
	assert.EqualError(t, err, "The context 'target-context' must match the current context to uninstall")
}

func TestUninstallOptions_Run_ContextSpecifiedAsOption_PassWhenContextNamesMatch(t *testing.T) {
	kubeMock := setupUninstall("correct-context-to-delete")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Kuber: kubeMock,
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
	kubeMock := setupUninstall("correct-context-to-delete")

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Kuber: kubeMock,
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

func TestUninstallOptions_Run_ContextSpecifiedViaCli_FailsWhenContextNamesDoNotMatch(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	kubeMock := setupUninstall("current-context")

	// mock terminal
	c, state, term := tests.NewTerminal(t)

	// Test interactive IO
	donec := make(chan struct{})
	go func() {
		defer close(donec)
		c.ExpectString("Enter the current context name to confirm uninstalllation of the Jenkins X platform from the ns namespace:")
		c.SendLine("target-context")
		c.ExpectEOF()
	}()

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Factory: cmd_test.NewMockFactory(),
			Kuber:   kubeMock,
			In:      term.In,
			Out:     term.Out,
			Err:     term.Err,
		},
		Namespace: "ns",
	}

	err := o.Run()
	assert.EqualError(t, err, "The context 'target-context' must match the current context to uninstall")

	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf(expect.StripTrailingEmptyLines(state.String()))
}

func TestUninstallOptions_Run_ContextSpecifiedViaCli_PassWhenContextNamesMatch(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	kubeMock := setupUninstall("correct-context-to-delete")

	// mock terminal
	c, state, term := tests.NewTerminal(t)

	// Test interactive IO
	donec := make(chan struct{})
	go func() {
		defer close(donec)
		c.ExpectString("Enter the current context name to confirm uninstalllation of the Jenkins X platform from the ns namespace:")
		c.SendLine("correct-context-to-delete")
		c.ExpectEOF()
	}()

	o := &cmd.UninstallOptions{
		CommonOptions: cmd.CommonOptions{
			Kuber: kubeMock,
			In:    term.In,
			Out:   term.Out,
			Err:   term.Err,
		},
		Namespace: "ns",
	}

	cmd.ConfigureTestOptions(&o.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	o.BatchMode = false // The above line sets batch mode to true. Set it back here :-(

	// Create fake namespace (that we will uninstall from)
	err := createNamespace(o, "ns")

	// Run the uninstall
	err = o.Run()
	assert.NoError(t, err)

	// Assert that the namespace has been deleted
	_, err = o.KubeClientCached.CoreV1().Namespaces().Get("ns", metav1.GetOptions{})
	assert.Error(t, err)

	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf(expect.StripTrailingEmptyLines(state.String()))
}

func createNamespace(o *cmd.UninstallOptions, ns string) error {
	_, err := o.KubeClientCached.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	return err
}
