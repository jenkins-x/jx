package cmd_test

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/prow/config"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/prow"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

const (
	protectionRepoName = "test-repo"
	protectionOrgName  = "test-org"
	protectionContext  = "test-context"
)

func TestStartProtection(t *testing.T) {
	o := cmd.StartProtectionOptions{
		CommonOptions: cmd.CommonOptions{},
	}

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		&gits.GitFake{},
		helm_test.NewMockHelmer(),
	)

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	assert.NoError(t, err)
	// First configure a repo in prow
	repo := fmt.Sprintf("%s/%s", protectionOrgName, protectionRepoName)
	repos := []string{repo}
	err = prow.AddApplication(kubeClient, repos, ns, "")
	defer func() {
		err = prow.DeleteApplication(kubeClient, repos, ns)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	o.Args = []string{protectionContext, repo}
	err = o.Run()
	assert.NoError(t, err)
	prowOptions := prow.Options{
		Kind:       config.Protection,
		KubeClient: kubeClient,
		NS:         ns,
	}
	prowConfig, _, err := prowOptions.GetProwConfig()
	assert.NoError(t, err)
	contexts, err := config.GetBranchProtectionContexts(protectionOrgName, protectionRepoName, prowConfig)
	assert.Equal(t, []string{protectionContext}, contexts)

}
