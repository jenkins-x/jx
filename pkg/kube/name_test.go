package kube

import (
	"github.com/stretchr/testify/assert"
	"testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


func TestGetName(t *testing.T) {
	r := &metav1.ObjectMeta{
		Name: "default-staging-my-spring-boot-demo2-my-spring-boot-demo2-fxfgz",
		Namespace: "default-staging",
		Labels: map[string]string{
			"app": "default-staging-my-spring-boot-demo2-my-spring-boot-demo2",
		},
	}
	assert.Equal(t, "my-spring-boot-demo2", GetName(r), "Get name on first pod")
}
