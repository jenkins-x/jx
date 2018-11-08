package common

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"k8s.io/client-go/kubernetes"
)

// OptionsInterface is an interface to allow passing around of a CommonOptions object without dependencies on the whole of the cmd package
type OptionsInterface interface {
	KubeClient() (kubernetes.Interface, string, error)
	VaultOperatorClient() (versioned.Interface, error)
	GetIn() terminal.FileReader
	GetOut() terminal.FileWriter
	GetErr() io.Writer
}
