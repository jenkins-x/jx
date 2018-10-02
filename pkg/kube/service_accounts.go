package kube

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
)

// PatchServiceAccount patches a given service account with a pull secret
func PatchServiceAccount(kubeClient, jxClient versioned.Interface, ns, pullSecret string) (*v1.Environment, error) {
	fmt.Printf("todo impl, pull secret is %s, namespace is %s", ns, pullSecret)
	return nil, nil
}
