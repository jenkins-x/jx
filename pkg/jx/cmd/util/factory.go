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
	"github.com/jenkins-x/jx/pkg/auth"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

const (
	jenkinsAuthConfigFile = "jenkinsAuth.yaml"
	GitAuthConfigFile     = "gitAuth.yaml"
)

type Factory interface {
	GetJenkinsClient() (*gojenkins.Jenkins, error)

	CreateAuthConfigService(fileName string) (auth.AuthConfigService, error)

	CreateGitAuthConfigService() (auth.AuthConfigService, error)

	CreateClient() (*kubernetes.Clientset, string, error)

	CreateJXClient() (*versioned.Clientset, string, error)

	CreateApiExtensionsClient() (*apiextensionsclientset.Clientset, error)

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

func (f *factory) CreateGitAuthConfigService() (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(GitAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	config, err := authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}

	// lets add a default if there's none defined yet
	if len(config.Servers) == 0 {
		config.Servers = []auth.AuthServer{
			{
				Name:  "GitHub",
				URL:   "github.com",
				Users: []auth.UserAuth{},
			},
		}
	}
	return authConfigSvc, nil
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

func (f *factory) CreateJXClient() (*versioned.Clientset, string, error) {
	kubeconfig := createKubeConfig()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := kube.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	ns := kube.CurrentNamespace(kubeConfig)
	client, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, ns, err
	}
	return client, ns, err

}

func (f *factory) CreateApiExtensionsClient() (*apiextensionsclientset.Clientset, error) {
	kubeconfig := createKubeConfig()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}

func (f *factory) CreateClient() (*kubernetes.Clientset, string, error) {
	kubeconfig := createKubeConfig()
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

var kubeConfigCache *string

func createKubeConfig() *string {
	var kubeconfig *string
	if kubeConfigCache != nil {
		return kubeConfigCache
	}
	kubeconfenv := os.Getenv("KUBECONFIG")
	if kubeconfenv != "" {
		kubeconfig = &kubeconfenv
	} else {
		if home := HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
	}
	kubeConfigCache = kubeconfig
	return kubeconfig
}

func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(os.Stdout)
}
