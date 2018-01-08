package util

import (
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"

	"github.com/jenkins-x/golang-jenkins"
	"k8s.io/client-go/kubernetes"
)

type Factory interface {
	GetJenkinsClient() (*gojenkins.Jenkins, error)

	CreateClient() (*kubernetes.Clientset, error)

	DefaultNamespace(client *kubernetes.Clientset) (string, error)

	CreateTable(out io.Writer) table.Table
}

type factory struct {
}

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory() Factory {
	return &factory{
	}
}

// GetJenkinsClient creates a new jenkins client
func (f *factory) GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("JENKINS_URL")
	if url == "" {
		// lets find the kubernets service
		client, err := f.CreateClient()
		if err != nil {
			return nil, err
		}
		ns, err := f.DefaultNamespace(client)
		if err != nil {
			return nil, err
		}
		url, err = kube.FindServiceURL(client, ns, "jenkins")
		if err != nil {
			return nil, err
		}
	}
	return jenkins.GetJenkinsClient(url)
}

func (*factory) CreateClient() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// TODO load from kubeconfig argument?
		//kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	return kube.CreateClient(kubeconfig)
}

func (*factory) DefaultNamespace(client *kubernetes.Clientset) (string, error) {
	// TODO
	return "jx", nil
}


func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(os.Stdout)
}
