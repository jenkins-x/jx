package cmd

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestJXNamespace(t *testing.T) {
	o := &CommonOptions{}
	configureTestOptions(o)

	kubeClient, ns, err := o.KubeClient()
	assert.NoError(t, err, "Failed to create kube client")

	if err == nil {
		resource, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to query namespace")
		if err == nil {
			fmt.Printf("Found namespace %#v\n", resource)
		}
	}

	_, err = o.CreateGitAuthConfigService()
	assert.NoError(t, err, "Failed to create GitAuthConfigService")
}
