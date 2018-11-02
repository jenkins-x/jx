package common

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type NewCommonOptionsInterface interface {
	KubeClient() (kubernetes.Interface, string, error)
	VaultOperatorClient() (versioned.Interface, error)
}
