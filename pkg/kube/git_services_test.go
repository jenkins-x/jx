package kube_test

import (
	"testing"

	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const serviceURL = "https://github.beescloud.com"
const secretName = kube.SecretJenkinsPipelineGitCredentials + "github-ghe"
const serviceKind = "github"

func createSecret(secretName string, labels map[string]string, annotations map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       secretName,
			DeletionGracePeriodSeconds: nil,
			Labels:                     labels,
			Annotations:                annotations,
		},
	}
}

func TestGitServiceKindFromSecrets(t *testing.T) {
	t.Parallel()
	secret := createSecret(secretName,
		map[string]string{
			"jenkins.io/service-kind": serviceKind,
		},
		map[string]string{
			"jenkins.io/url": serviceURL,
		})
	kubeClient := fake.NewSimpleClientset(secret)

	foundServiceKind, err := kube.GetServiceKindFromSecrets(kubeClient, "", serviceURL)

	assert.NoError(t, err, "should find a service kind without any error")
	assert.Equal(t, serviceKind, foundServiceKind, "should find a service kind equal with '%s'", serviceKind)
}

func TestGitServiceKindFromSecretsWithMissingKind(t *testing.T) {
	t.Parallel()
	secret := createSecret("jx-pipeline-git",
		map[string]string{
			"jenkins.io/kind":         "git",
			"jenkins.io/service-kind": "",
		},
		map[string]string{
			"jenkins.io/url": serviceURL,
		})
	ns := "jx"
	secret.Namespace = ns
	kubeClient := fake.NewSimpleClientset(secret)
	jxClient := v1fake.NewSimpleClientset()

	foundServiceKind, err := kube.GetGitServiceKind(jxClient, kubeClient, ns, serviceURL)

	t.Logf("found service kind %s for URL %s\n\n", foundServiceKind, serviceURL)

	assert.NoError(t, err, "should find a service kind without any error")
	assert.Equal(t, serviceKind, foundServiceKind, "should find a service kind equal with '%s'", serviceKind)
}

func TestGitServiceKindFromSecretsWithoutURL(t *testing.T) {
	t.Parallel()
	secret := createSecret(secretName,
		map[string]string{
			"jenkins.io/service-kind": serviceKind,
		},
		nil)

	kubeClient := fake.NewSimpleClientset(secret)
	foundServiceKind, err := kube.GetServiceKindFromSecrets(kubeClient, "", serviceURL)

	assert.Error(t, err, "should not found a service kind")
	assert.Equal(t, "", foundServiceKind, "should return no service kind")
}

func TestGitServiceKindFromSecretsWithoutKindLabel(t *testing.T) {
	t.Parallel()
	secret := createSecret(secretName,
		map[string]string{
			"jenkins.io/service-kind": "test",
		},
		map[string]string{
			"jenkins.io/url": "test",
		})
	kubeClient := fake.NewSimpleClientset(secret)

	foundServiceKind, err := kube.GetServiceKindFromSecrets(kubeClient, "", serviceURL)

	assert.Error(t, err, "should not found a service kind")
	assert.Equal(t, "", foundServiceKind, "should return no service kind")
}
