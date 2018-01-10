package util

import (
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/golang-jenkins"
	"k8s.io/client-go/kubernetes"
	"github.com/jenkins-x/jx/pkg/auth"
)

const (
	jenkinsAuthConfigFile = "jenkinsAuth.yaml"
	GitAuthConfigFile = "gitAuth.yaml"
)

type Factory interface {
	GetJenkinsClient() (*gojenkins.Jenkins, error)

	CreateAuthConfigService(fileName string) (auth.AuthConfigService, error)

	CreateClient() (*kubernetes.Clientset, string, error)

	CreateTable(out io.Writer) table.Table
}

type factory struct {
	Batch bool
}

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory() Factory {
	return &factory{}
}

// GetJenkinsClient creates a new jenkins client
func (f *factory) GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("JENKINS_URL")
	if url == "" {
		// lets find the kubernets service
		client, ns, err := f.CreateClient()
		if err != nil {
			return nil, err
		}
		url, err = kube.FindServiceURL(client, ns, kube.ServiceJenkins)
		if err != nil {
			return nil, err
		}
	}
	svc, err := f.CreateAuthConfigService(jenkinsAuthConfigFile)
	if err != nil {
		return nil, err
	}
	return jenkins.GetJenkinsClient(url, f.Batch, &svc)
}

func (f *factory) CreateAuthConfigService(fileName string) (auth.AuthConfigService, error) {
	svc := auth.AuthConfigService{}
	dir, err := ConfigDir()
	if err != nil {
		return svc, err
	}
	svc.FileName = filepath.Join(dir, fileName)
	return svc, nil
}

func (f *factory) CreateClient() (*kubernetes.Clientset, string, error) {
	var kubeconfig *string
	kubeconfenv := os.Getenv("KUBECONFIG")
	if kubeconfenv != "" {
		kubeconfig = &kubeconfenv
	} else {
		if home := HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			// TODO load from kubeconfig CLI option?
			//kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
	}
	client, err := kube.CreateClient(kubeconfig)
	if err != nil {
		return nil, "", nil
	}
	ns := ""
	config, _, err := kube.LoadConfig()
	if err != nil {
		return client, ns, err
	}
	ns = kube.CurrentNamespace(config)
	// TODO allow namsepace to be specified as a CLI argument!
	return client, ns, nil
}

func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(os.Stdout)
}
