package kube

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGitServiceKinds(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigMapJenkinsXGitKinds,
		},
		Data: map[string]string{
			"github": "foo: https://github.acme.com\nbar: https://github.bar.com/api/v3",
			"gitea":  "cheese: https://gitea.cheese.com",
		},
	}

	assertGitKind := func(kind, url string) {
		actual := GetGitServiceKindFromConfigMap(cm, url)
		assert.Equal(t, kind, actual, "Get kind for URL %s", url)
	}

	assertGitKind("github", "https://github.acme.com")
	assertGitKind("github", "https://github.acme.com/")
	assertGitKind("github", "https://github.bar.com")
	assertGitKind("github", "https://github.bar.com/")
	assertGitKind("gitea", "https://gitea.cheese.com")
}
