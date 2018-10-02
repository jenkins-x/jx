package kube

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
)

// EnsureDevEnvironmentSetup ensures that the Environment is created in the given namespace
func PatchServiceAccount(kubeClient, jxClient versioned.Interface, ns, pullSecretsInput string) (*v1.Environment, error) {

	return nil, nil
}
