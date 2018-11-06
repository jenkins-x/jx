package common

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"k8s.io/client-go/kubernetes"
)

type NewCommonOptionsInterface interface {
	KubeClient() (kubernetes.Interface, string, error)
	VaultOperatorClient() (versioned.Interface, error)
	GetIn() terminal.FileReader
	GetOut() terminal.FileWriter
	GetErr() io.Writer
}
