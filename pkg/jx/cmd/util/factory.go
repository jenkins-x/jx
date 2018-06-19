package util

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

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

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AddonAuthConfigFile       = "addonAuth.yaml"
	JenkinsAuthConfigFile     = "jenkinsAuth.yaml"
	IssuesAuthConfigFile      = "issuesAuth.yaml"
	ChatAuthConfigFile        = "chatAuth.yaml"
	GitAuthConfigFile         = "gitAuth.yaml"
	ChartmuseumAuthConfigFile = "chartmuseumAuth.yaml"
)

type Factory interface {
	CreateJenkinsClient(kubeClient kubernetes.Interface, ns string) (*gojenkins.Jenkins, error)

	GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error)

	CreateAuthConfigService(fileName string) (auth.AuthConfigService, error)

	CreateJenkinsAuthConfigService() (auth.AuthConfigService, error)

	CreateChartmuseumAuthConfigService() (auth.AuthConfigService, error)

	CreateIssueTrackerAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error)

	CreateChatAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error)

	CreateAddonAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error)

	CreateClient() (kubernetes.Interface, string, error)

	CreateKubeConfig() (*rest.Config, error)

	CreateJXClient() (versioned.Interface, string, error)

	CreateApiExtensionsClient() (apiextensionsclientset.Interface, error)

	CreateMetricsClient() (*metricsclient.Clientset, error)

	CreateComplianceClient() (*client.SonobuoyClient, error)

	CreateTable(out io.Writer) table.Table

	SetBatch(batch bool)

	ImpersonateUser(user string) Factory

	IsInCluster() bool

	IsInCDPIpeline() bool

	AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error
}

type factory struct {
	Batch bool

	impersonateUser string
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

// ImpersonateUser returns a new factory impersonating the given user
func (f *factory) ImpersonateUser(user string) Factory {
	copy := *f
	copy.impersonateUser = user
	return &copy
}

// CreateJenkinsClient creates a new jenkins client
func (f *factory) CreateJenkinsClient(kubeClient kubernetes.Interface, ns string) (*gojenkins.Jenkins, error) {

	svc, err := f.CreateJenkinsAuthConfigService()
	if err != nil {
		return nil, err
	}
	url, err := f.GetJenkinsURL(kubeClient, ns)
	if err != nil {
		return nil, fmt.Errorf("%s. Try switching to the Development Tools environment via: jx env dev", err)
	}
	return jenkins.GetJenkinsClient(url, f.Batch, &svc)
}

func (f *factory) GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error) {
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
				return "", fmt.Errorf("%s in namespaces %s and %s", err, realNS, ns)
			}
		}
	}
	if err != nil {
		return "", fmt.Errorf("%s in namespace %s", err, ns)
	}
	return url, err
}

func (f *factory) CreateJenkinsAuthConfigService() (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(JenkinsAuthConfigFile)

	if err != nil {
		return authConfigSvc, err
	}
	config, err := authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}

	if len(config.Servers) == 0 {
		c, ns, err := f.CreateClient()
		if err != nil {
			return authConfigSvc, err
		}

		s, err := c.CoreV1().Secrets(ns).Get(kube.SecretJenkins, metav1.GetOptions{})
		if err != nil {
			return authConfigSvc, err
		}

		userAuth := auth.UserAuth{
			Username: "admin",
			ApiToken: string(s.Data[kube.JenkinsAdminApiToken]),
		}
		svc, err := c.CoreV1().Services(ns).Get(kube.ServiceJenkins, metav1.GetOptions{})
		if err != nil {
			return authConfigSvc, err
		}
		svcURL := kube.GetServiceURL(svc)
		if svcURL == "" {
			return authConfigSvc, fmt.Errorf("unable to find external URL annotation on service %s", svc.Name)
		}

		u, err := url.Parse(svcURL)
		if err != nil {
			return authConfigSvc, err
		}
		if !userAuth.IsInvalid() {
			config.Servers = []*auth.AuthServer{
				{
					Name:  u.Host,
					URL:   svcURL,
					Users: []*auth.UserAuth{&userAuth},
				},
			}
		}
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
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindIssue, f.IsInCDPIpeline())
	}
	return authConfigSvc, err
}

func (f *factory) CreateChatAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(ChatAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindChat, f.IsInCDPIpeline())
	}
	return authConfigSvc, err
}

func (f *factory) CreateAddonAuthConfigService(secrets *corev1.SecretList) (auth.AuthConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(AddonAuthConfigFile)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindChat, f.IsInCDPIpeline())
	}
	return authConfigSvc, err
}

func (f *factory) AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error {
	if config == nil || secrets == nil {
		return nil
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
					// lets use the latest values from the credential
					if k != "" {
						server.Kind = k
					}
					if name != "" {
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
	return nil
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

func (f *factory) CreateJXClient() (versioned.Interface, string, error) {
	config, err := f.CreateKubeConfig()
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

func (f *factory) CreateApiExtensionsClient() (apiextensionsclientset.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}

func (f *factory) CreateMetricsClient() (*metricsclient.Clientset, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return metricsclient.NewForConfig(config)
}

func (f *factory) CreateClient() (kubernetes.Interface, string, error) {
	cfg, err := f.CreateKubeConfig()
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

func (f *factory) CreateKubeConfig() (*rest.Config, error) {
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
	user := f.getImpersonateUser()
	if config != nil && user != "" && config.Impersonate.UserName == "" {
		config.Impersonate.UserName = user
	}
	return config, nil
}

func (f *factory) getImpersonateUser() string {
	user := f.impersonateUser
	if user == "" {
		// this is really only used for testing really
		user = os.Getenv("JX_IMPERSONATE_USER")
	}
	return user
}

func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(os.Stdout)
}

// IsInCDPIpeline we should only load the git / issue tracker API tokens if the current pod
// is in a pipeline and running as the jenkins service account
func (f *factory) IsInCDPIpeline() bool {
	// TODO should we let RBAC decide if we can see the Secrets in the dev namespace?
	// or we should test if we are in the cluster and get the current ServiceAccount name?
	return os.Getenv("BUILD_NUMBER") != ""
}

// function to tell if we are running incluster
func (f *factory) IsInCluster() bool {
	_, err := rest.InClusterConfig()
	if err != nil {
		return false
	}
	return true
}

// CreateComplianceClient creates a new Sonobuoy compliance client
func (f *factory) CreateComplianceClient() (*client.SonobuoyClient, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "compliance client failed to load the Kubernetes configuration")
	}
	return client.NewSonobuoyClient(config)
}
