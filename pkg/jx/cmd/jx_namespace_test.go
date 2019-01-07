package cmd_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestJXNamespace(t *testing.T) {
	t.Parallel()
	o := &cmd.CommonOptions{}
	cmd.ConfigureTestOptions(o, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, "", true))

	kubeClient, ns, err := o.KubeClientAndNamespace()
	assert.NoError(t, err, "Failed to create kube client")

	if err == nil {
		resource, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to query namespace")
		if err == nil {
			log.Warnf("Found namespace %#v\n", resource)
		}
	}

	_, err = o.CreateGitAuthConfigService()
	assert.NoError(t, err, "Failed to create GitAuthConfigService")
}
