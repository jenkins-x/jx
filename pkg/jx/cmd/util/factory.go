package util

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metricsclient "k8s.io/metrics/pkg/client/clientset_generated/clientset"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	"github.com/jenkins-x/jx/pkg/gits"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JenkinsAuthConfigFile     = "jenkinsAuth.yaml"
	IssuesAuthConfigFile      = "issuesAuth.yaml"
	GitAuthConfigFile         = "gitAuth.yaml"
	ChartmuseumAuthConfigFile = "chartmuseumAuth.yaml"
)

type Factory interface {
	CreateJenkinsClient() (*gojenkins.Jenkins, error)

	GetJenkinsURL() (string, error)

	CreateAuthConfigService(fileName string) (auth.AuthConfigService, error)

	CreateGitAuthConfigService() (auth.AuthConfigService, error)

	CreateJenkinsAuthConfigService() (auth.AuthConfigService, error)

	CreateChartmuseumAuthConfigService() (auth.AuthConfigService, error)

	CreateIssueTrackerAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error)

	CreateClient() (*kubernetes.Clientset, string, error)

	CreateJXClient() (*versioned.Clientset, string, error)

	CreateApiExtensionsClient() (*apiextensionsclientset.Clientset, error)

	CreateMetricsClient() (*metricsclient.Clientset, error)

	CreateTable(out io.Writer) table.Table

	SetBatch(batch bool)

	LoadPipelineSecrets(kind string) (*corev1.SecretList, error)
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

func (f *factory) SetBatch(batch bool) {
	f.Batch = batch
}

// CreateJenkinsClient creates a new jenkins client
func (f *factory) CreateJenkinsClient() (*gojenkins.Jenkins, error) {

	svc, err := f.CreateJenkinsAuthConfigService()
	if err != nil {
		return nil, err
	}
	url, err := f.GetJenkinsURL()
	if err != nil {
		return nil, err
	}
	return jenkins.GetJenkinsClient(url, f.Batch, &svc)
}

func (f *factory) GetJenkinsURL() (string, error) {
	// lets find the kubernetes service
	client, ns, err := f.CreateClient()
	if err != nil {
		return "", err
	}
	url, err := kube.FindServiceURL(client, ns, kube.ServiceJenkins)
	if err != nil {
		// lets try the real environment
		realNS, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return "", err
		}
		if realNS != ns {
			url, err = kube.FindServiceURL(client, realNS, kube.ServiceJenkins)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return url, err
}

func (f *factory) CreateJenkinsAuthConfigService() (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(JenkinsAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	_, err = authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}
	return authConfigSvc, err
}

func (f *factory) CreateChartmuseumAuthConfigService() (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(ChartmuseumAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	_, err = authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}
	return authConfigSvc, err
}

func (f *factory) CreateIssueTrackerAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(IssuesAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.authMergePipelineSecrets(config, secrets, kube.ValueKindIssue, f.isInCDPIpeline())
	}
	return authConfigSvc, err
}

func (f *factory) authMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) {
	if config == nil || secrets == nil {
		return
	}
	for _, secret := range secrets.Items {
		labels := secret.Labels
		annotations := secret.Annotations
		data := secret.Data
		if labels != nil && labels[kube.LabelKind] == kind && annotations != nil {
			u := annotations[kube.AnnotationURL]
			name := annotations[kube.AnnotationName]
			k := labels[kube.LabelServiceKind]
			if u != "" {
				server := config.GetOrCreateServer(u)
				if server != nil {
					if server.Kind == "" {
						server.Kind = k
					}
					if server.Name == "" {
						server.Name = name
					}
					if data != nil {
						username := data[kube.SecretDataUsername]
						pwd := data[kube.SecretDataPassword]
						if len(username) > 0 && isCDPipeline {
							userAuth := config.GetOrCreateUserAuth(u, string(username))
							if userAuth != nil {
								if len(pwd) > 0 {
									userAuth.ApiToken = string(pwd)
								}
							}
						}
					}
				}
			}
		}
	}
}

func (f *factory) CreateGitAuthConfigService() (auth.AuthConfigService, error) {
	secrets, err := f.LoadPipelineSecrets(kube.ValueKindGit)
	if err != nil {

		kubeConfig, _, configLoadErr := kube.LoadConfig()
		if err != nil {
			fmt.Printf("WARNING: Could not load config: %s", configLoadErr)
		}

		ns := kube.CurrentNamespace(kubeConfig)
		if ns == "" {
			fmt.Printf("WARNING: Could not get the current namespace")
		}

		fmt.Printf("WARNING: The current user cannot query secrets in the namespace %s: %s\n", ns, err)
	}

	fileName := GitAuthConfigFile
	return f.createGitAuthConfigServiceFromSecrets(fileName, secrets, f.isInCDPIpeline())
}

func (f *factory) createGitAuthConfigServiceFromSecrets(fileName string, secrets *corev1.SecretList, isCDPipeline bool) (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(fileName)
	if err != nil {
		return authConfigSvc, err
	}

	config, err := authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}

	// lets add a default if there's none defined yet
	if len(config.Servers) == 0 {
		// if in cluster then there's no user configfile, so check for env vars first
		userAuth := auth.CreateAuthUserFromEnvironment("GIT")
		if !userAuth.IsInvalid() {
			// if no config file is being used lets grab the git server from the current directory
			server, err := gits.GetGitServer("")
			if err != nil {
				fmt.Printf("WARNING: unable to get remote git repo server, %v\n", err)
				server = "https://github.com"
			}
			config.Servers = []*auth.AuthServer{
				{
					Name:  "Git",
					URL:   server,
					Users: []*auth.UserAuth{&userAuth},
				},
			}
		}
	}

	if len(config.Servers) == 0 {
		config.Servers = []*auth.AuthServer{
			{
				Name:  "GitHub",
				URL:   "https://github.com",
				Kind:  gits.KindGitHub,
				Users: []*auth.UserAuth{},
			},
		}
	}

	if secrets != nil {
		f.authMergePipelineSecrets(config, secrets, kube.ValueKindGit, isCDPipeline)
	}
	return authConfigSvc, nil
}

func (f *factory) LoadPipelineSecrets(kind string) (*corev1.SecretList, error) {
	kubeClient, curNs, err := f.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a kuberntees client %s", err)
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, curNs)
	if err != nil {
		return nil, fmt.Errorf("Failed to get the development environment %s", err)
	}
	// TODO use kind as a label selector...
	return kubeClient.CoreV1().Secrets(ns).List(metav1.ListOptions{})
}

func (f *factory) mergePipeineSecrets(config *auth.AuthConfig, secretList *corev1.SecretList) {

}

func (f *factory) CreateAuthConfigService(fileName string) (auth.AuthConfigService, error) {
	svc := auth.AuthConfigService{}
	dir, err := util.ConfigDir()
	if err != nil {
		return svc, err
	}
	svc.FileName = filepath.Join(dir, fileName)
	return svc, nil
}

func (f *factory) CreateJXClient() (*versioned.Clientset, string, error) {
	config, err := f.createKubeConfig()
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
	config, err := f.createKubeConfig()
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}

func (f *factory) CreateMetricsClient() (*metricsclient.Clientset, error) {
	config, err := f.createKubeConfig()
	if err != nil {
		return nil, err
	}
	return metricsclient.NewForConfig(config)
}

func (f *factory) CreateClient() (*kubernetes.Clientset, string, error) {
	cfg, err := f.createKubeConfig()
	if err != nil {
		return nil, "", err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, "", err
	}
	if client == nil {
		return nil, "", fmt.Errorf("Failed to create Kubernetes Client!")
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
		if home := util.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
	}
	kubeConfigCache = kubeconfig
	return kubeconfig
}

func (f *factory) createKubeConfig() (*rest.Config, error) {
	kubeconfig := createKubeConfig()
	var config *rest.Config
	var err error
	if kubeconfig != nil {
		exists, err := util.FileExists(*kubeconfig)
		if err == nil && exists {
			// use the current context in kubeconfig
			config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				return nil, err
			}
		}
	}
	if config == nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(os.Stdout)
}

// isInCDPIpeline we should only load the git / issue tracker API tokens if the current pod
// is in a pipeline and running as the jenkins service account
func (f *factory) isInCDPIpeline() bool {
	// TODO should we let RBAC decide if we can see the Secrets in the dev namespace?
	// or we should test if we are in the cluster and get the current ServiceAccount name?
	return os.Getenv("BUILD_NUMBER") != ""
}
