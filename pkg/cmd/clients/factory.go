package clients

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/vault"
	certmngclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube/services"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	build "github.com/knative/build/pkg/client/clientset/versioned"
	kserve "github.com/knative/serving/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type factory struct {
	Batch bool

	kubeConfig      kube.Kuber
	kubeConfigCache *string
	impersonateUser string
	bearerToken     string
	secretLocation  secrets.SecretLocation
	offline         bool
}

var _ Factory = (*factory)(nil)

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory() Factory {
	f := &factory{}
	f.kubeConfig = kube.NewKubeConfig()
	return f
}

func (f *factory) SetBatch(batch bool) {
	f.Batch = batch
}

func (f *factory) SetOffline(offline bool) {
	f.offline = offline
}

// ImpersonateUser returns a new factory impersonating the given user
func (f *factory) ImpersonateUser(user string) Factory {
	copy := *f
	copy.impersonateUser = user
	return &copy
}

// WithBearerToken returns a new factory with bearer token
func (f *factory) WithBearerToken(token string) Factory {
	copy := *f
	copy.bearerToken = token
	return &copy
}

// CreateJenkinsClient creates a new Jenkins client
func (f *factory) CreateJenkinsClient(kubeClient kubernetes.Interface, ns string) (gojenkins.JenkinsClient, error) {
	svc, err := f.CreateJenkinsConfigService(auth.AutoConfigKind)
	if err != nil {
		return nil, err
	}
	url, err := f.GetJenkinsURL(kubeClient, ns)
	if err != nil {
		return nil, fmt.Errorf("%s. Try switching to the Development Tools environment via: jx env dev", err)
	}
	cfg, err := svc.Config()
	if err != nil {
		return nil, err
	}
	server, err := cfg.GetServer(url)
	if err != nil {
		return nil, err
	}
	return jenkins.GetJenkinsClient(server)
}

// CreateCustomJenkinsClient creates a new Jenkins client for the given custom Jenkins App
func (f *factory) CreateCustomJenkinsClient(kubeClient kubernetes.Interface, ns string,
	jenkinsServiceName string) (gojenkins.JenkinsClient, error) {
	svc, err := f.CreateJenkinsConfigService(auth.AutoConfigKind)
	if err != nil {
		return nil, err
	}
	url, err := f.GetCustomJenkinsURL(kubeClient, ns, jenkinsServiceName)
	if err != nil {
		return nil, fmt.Errorf("%s. Try switching to the Development Tools environment via: jx env dev", err)
	}
	cfg, err := svc.Config()
	if err != nil {
		return nil, err
	}
	server, err := cfg.GetServer(url)
	if err != nil {
		return nil, err
	}
	return jenkins.GetJenkinsClient(server)
}

// GetJenkinsURL gets the Jenkins URL for the given namespace
func (f *factory) GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error) {
	// lets find the Kubernetes service
	client, curNS, err := f.CreateKubeClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to create the kube client")
	}
	if ns == "" {
		ns = curNS
	}
	url, err := services.FindServiceURL(client, ns, kube.ServiceJenkins)
	if err != nil {
		// lets try the real environment
		realNS, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get the dev namespace from '%s' namespace", ns)
		}
		if realNS != ns {
			url, err = services.FindServiceURL(client, realNS, kube.ServiceJenkins)
			if err != nil {
				return "", fmt.Errorf("%s in namespaces %s and %s", err, realNS, ns)
			}
			return url, nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("%s in namespace %s", err, ns)
	}
	return url, err
}

// GetCustomJenkinsURL gets a custom jenkins App service URL
func (f *factory) GetCustomJenkinsURL(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string) (string, error) {
	// lets find the Kubernetes service
	client, curNs, err := f.CreateKubeClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to create the kube client")
	}
	if ns == "" {
		ns = curNs
	}
	url, err := services.FindServiceURL(client, ns, jenkinsServiceName)
	if err != nil {
		// lets try the real environment
		realNS, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get the dev namespace from '%s' namespace", ns)
		}
		if realNS != ns {
			url, err = services.FindServiceURL(client, realNS, jenkinsServiceName)
			if err != nil {
				return "", errors.Wrapf(err, "failed to find service URL for %s in namespaces %s and %s", jenkinsServiceName, realNS, ns)
			}
			return url, nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("%s in namespace %s", err, ns)
	}
	return url, err
}

// CreateJenkinsConfigService a jenkins config service whihc is able to load/save the jenkins configuration
func (f *factory) CreateJenkinsConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.JenkinsAuthConfigFile, kube.ValueKindJenkins, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for jenkins")
	}

	return cs, nil
}

// CreateChartmuseumConfigService  a chartmuseum  config service which is able to load/save the chartmuseum configuration
func (f *factory) CreateChartmuseumConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.ChartmuseumAuthConfigFile, kube.ValueKindChartmuseum, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for chartmuseum")
	}
	return cs, nil
}

// CreateIssueTrackerConfigService  a issue tracker config service which is able to load/save the issue tracker configuration
func (f *factory) CreateIssueTrackerConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.IssuesAuthConfigFile, kube.ValueKindIssue, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for tracker")
	}
	return cs, nil
}

// CreateChatConfigService a chat config service which is able to load/save the chat configuration
func (f *factory) CreateChatConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.ChatAuthConfigFile, kube.ValueKindChat, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for chat")
	}
	return cs, nil
}

// CreateAddonConfigServicecreates a addon config service which is able to load/save the addon configuration
func (f *factory) CreateAddonConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.AddonAuthConfigFile, kube.ValueKindAddon, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for addon")
	}
	return cs, nil
}

// CreateGitConfigService creates a git config service which is able to load/save the git configuration
func (f *factory) CreateGitConfigService(kind auth.ConfigKind) (auth.ConfigService, error) {
	cs, err := f.CreateConfigService(kind, auth.GitAuthConfigFile, kube.ValueKindGit, "")
	if err != nil {
		return nil, errors.Wrap(err, "creating config service for git")
	}
	return cs, nil
}

// CreateAuthConfigService creates a new auth config service for the provided server and services. The config service location is read from
// configuration. It could be one of: vault, k8s secrets, local file-system.
func (f *factory) CreateConfigService(kind auth.ConfigKind, fileName string, serverKind string, serviceKind string) (auth.ConfigService, error) {
	switch kind {
	case auth.LocalConfigKind:
		return auth.NewFileConfigService(fileName)
	case auth.PipelineConfigKind:
		return f.createPipelineConfigService(fileName, serverKind, serviceKind)
	case auth.AutoConfigKind:
		if f.IsInCluster() {
			return f.createPipelineConfigService(fileName, serverKind, serviceKind)
		} else {
			return auth.NewFileConfigService(fileName)
		}
	default:
		return nil, fmt.Errorf("invalid config location %q", kind)
	}
}

func (f *factory) createPipelineConfigService(fileName string, serverKind string, serviceKind string) (auth.ConfigService, error) {
	client, namespace, err := f.CreateKubeClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating kubernetes client")
	}
	location := f.SecretsLocation()
	switch location {
	case secrets.VaultLocationKind:
		vaultClient, err := f.CreateSystemVaultClient(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "creating system vault")
		}
		cs := auth.NewVaultConfigService(fileName, vaultClient)
		return cs, nil
	case secrets.KubeLocationKind:
		devNs, _, err := kube.GetDevNamespace(client, namespace)
		if err != nil {
			return nil, errors.Wrap(err, "getting the dev namesapce")
		}
		cs := auth.NewKubeConfigService(client, devNs, serverKind, serviceKind)
		return cs, nil
	case secrets.FileSystemLocationKind:
		return auth.NewFileConfigService(fileName)
	default:
		return nil, fmt.Errorf("invalid location %q for pipeline secrets", location)
	}
}

// CreateGitProvider creates a new Git provider from given git URL
func (f *factory) CreateGitProvider(gitURL string, kind auth.ConfigKind, gitter gits.Gitter) (gits.GitProvider, error) {
	repository, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	cs, err := f.CreateGitConfigService(kind)
	if err != nil {
		return nil, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return nil, err
	}
	server, err := cfg.GetServer(repository.HostURL())
	if err != nil {
		return nil, err
	}
	return gits.CreateProvider(server, gitter)
}

// SecretsLocation indicates the location where the secrets are stored
func (f *factory) SecretsLocation() secrets.SecretsLocationKind {
	client, namespace, err := f.CreateKubeClient()
	if err != nil {
		return secrets.FileSystemLocationKind
	}
	if f.secretLocation == nil {
		devNs, _, err := kube.GetDevNamespace(client, namespace)
		if err != nil {
			devNs = kube.DefaultNamespace
		}
		f.secretLocation = secrets.NewSecretLocation(client, devNs)
	}
	return f.secretLocation.Location()
}

// SetSecretsLocation configures the secrets location. It will persist the value in a config map
// if the persist flag is set.
func (f *factory) SetSecretsLocation(location secrets.SecretsLocationKind, persist bool) error {
	if f.secretLocation == nil {
		client, namespace, err := f.CreateKubeClient()
		if err != nil {
			return errors.Wrap(err, "creating the kube client")
		}
		f.secretLocation = secrets.NewSecretLocation(client, namespace)
	}
	err := f.secretLocation.SetLocation(location, persist)
	if err != nil {
		return errors.Wrapf(err, "setting the secrets location %q", location)
	}
	return nil
}

// ResetSecretsLocation resets the location of the secrets stored in memory
func (f *factory) ResetSecretsLocation() {
	f.secretLocation = nil
}

// CreateSystemVaultClient gets the system vault client for managing the secrets
func (f *factory) CreateSystemVaultClient(namespace string) (vault.Client, error) {
	name, err := f.getVaultName(namespace)
	if err != nil {
		return nil, err
	}
	return f.CreateVaultClient(name, namespace)
}

// getVaultName gets the vault name from install configuration or builds a new name from
// cluster name
func (f *factory) getVaultName(namespace string) (string, error) {
	kubeClient, _, err := f.CreateKubeClient()
	if err != nil {
		return "", err
	}
	var name string
	if data, err := kube.ReadInstallValues(kubeClient, namespace); err == nil && data != nil {
		name = data[kube.SystemVaultName]
		if name == "" {
			clusterName := data[kube.ClusterName]
			if clusterName != "" {
				name = kubevault.SystemVaultNameForCluster(clusterName)
			}
		}
	}

	if name == "" {
		name, err = kubevault.SystemVaultName(f.kubeConfig)
		if err != nil || name == "" {
			return name, fmt.Errorf("could not find the system vault name in namespace %q", namespace)
		}
	}
	return name, nil
}

// CreateVaultClient returns the given vault client for managing secrets
// Will use default values for name and namespace if nil values are applied
func (f *factory) CreateVaultClient(name string, namespace string) (vault.Client, error) {
	vopClient, err := f.CreateVaultOperatorClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating the vault operator client")
	}
	kubeClient, defaultNamespace, err := f.CreateKubeClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating the kube client")
	}

	// Use the dev namespace from default namespace if nothing is specified by the user
	if namespace == "" {
		devNamespace, _, err := kube.GetDevNamespace(kubeClient, defaultNamespace)
		if err != nil {
			return nil, errors.Wrapf(err, "getting the dev namespace from current namespace %q",
				defaultNamespace)
		}
		namespace = devNamespace
	}

	// Get the system vault name from configuration if nothing is specified by the user
	if name == "" {
		name, err = f.getVaultName(namespace)
		if err != nil || name == "" {
			return nil, errors.Wrap(err, "reading the vault name from configuration")
		}
	}

	if !kubevault.FindVault(vopClient, name, namespace) {
		return nil, fmt.Errorf("no %q vault found in namespace %q", name, namespace)
	}

	clientFactory, err := kubevault.NewVaultClientFactory(kubeClient, vopClient, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "creating vault client")
	}
	vaultClient, err := clientFactory.NewVaultClient(name, namespace)
	return vault.NewVaultClient(vaultClient), err
}

// CreateJXClient creates a new JX client
func (f *factory) CreateJXClient() (versioned.Interface, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}

	kubeConfig, _, err := f.kubeConfig.LoadConfig()
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

// CreateKnativeBuildClient creates a new knative build client
func (f *factory) CreateKnativeBuildClient() (build.Interface, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.kubeConfig.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	ns := kube.CurrentNamespace(kubeConfig)
	client, err := build.NewForConfig(config)
	if err != nil {
		return nil, ns, err
	}
	return client, ns, err
}

func (f *factory) CreateKnativeServeClient() (kserve.Interface, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.kubeConfig.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	ns := kube.CurrentNamespace(kubeConfig)
	client, err := kserve.NewForConfig(config)
	if err != nil {
		return nil, ns, err
	}
	return client, ns, err
}

// Create a new Tekton client
func (f *factory) CreateTektonClient() (tektonclient.Interface, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.kubeConfig.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	ns := kube.CurrentNamespace(kubeConfig)
	client, err := tektonclient.NewForConfig(config)
	if err != nil {
		return nil, ns, err
	}
	return client, ns, err
}

// CreateDynamicClient creates a enw dynamic client
func (f *factory) CreateDynamicClient() (*dynamic.APIHelper, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.kubeConfig.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	ns := kube.CurrentNamespace(kubeConfig)
	client, err := dynamic.NewAPIHelperFromRESTConfig(config)
	if err != nil {
		return nil, ns, err
	}
	return client, ns, err
}

// CreateApiExtensionsClient creates a new API extensions client
func (f *factory) CreateApiExtensionsClient() (apiextensionsclientset.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return apiextensionsclientset.NewForConfig(config)
}

// CreateMetricsClient creates a new metrics client
func (f *factory) CreateMetricsClient() (*metricsclient.Clientset, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return metricsclient.NewForConfig(config)
}

// CreateKubeClients creates a new Kubernetes client
func (f *factory) CreateKubeClient() (kubernetes.Interface, string, error) {
	cfg, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, "", err
	}
	if client == nil {
		return nil, "", fmt.Errorf("Failed to create Kubernetes Client")
	}
	ns := ""
	config, _, err := f.kubeConfig.LoadConfig()
	if err != nil {
		return client, ns, err
	}
	ns = kube.CurrentNamespace(config)
	// TODO allow namsepace to be specified as a CLI argument!
	return client, ns, nil
}

func (f *factory) createKubeConfig(offline bool) *string {
	if offline {
		panic("not supposed to be making a network connection")
	}
	var kubeconfig *string
	if f.kubeConfigCache != nil {
		return f.kubeConfigCache
	}
	if home := util.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	f.kubeConfigCache = kubeconfig
	return kubeconfig
}

// CreateKubeConfig creates the kube configuration
func (f *factory) CreateKubeConfig() (*rest.Config, error) {
	masterURL := ""
	kubeConfigEnv := os.Getenv("KUBECONFIG")
	if kubeConfigEnv != "" {
		pathList := filepath.SplitList(kubeConfigEnv)
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{Precedence: pathList},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterURL}}).ClientConfig()
	}
	kubeconfig := f.createKubeConfig(f.offline)
	var config *rest.Config
	var err error
	if kubeconfig != nil {
		exists, err := util.FileExists(*kubeconfig)
		if err == nil && exists {
			// use the current context in kubeconfig
			config, err = clientcmd.BuildConfigFromFlags(masterURL, *kubeconfig)
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

	if config != nil && f.bearerToken != "" {
		config.BearerToken = f.bearerToken
		return config, nil
	}

	user := f.getImpersonateUser()
	if config != nil && user != "" && config.Impersonate.UserName == "" {
		config.Impersonate.UserName = user
	}

	// for testing purposes one can enable tracing of Kube REST API calls
	trace := os.Getenv("TRACE_KUBE_API")
	if trace == "1" || trace == "on" {
		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &Tracer{rt}
		}
	}
	return config, nil
}

// getImpersonateUser retruns the impersonated user
func (f *factory) getImpersonateUser() string {
	user := f.impersonateUser
	if user == "" {
		// this is really only used for testing really
		user = os.Getenv("JX_IMPERSONATE_USER")
	}
	return user
}

// CreateTable creates a new table
func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(out)
}

// function to tell if we are running incluster
func (f *factory) IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err != nil
}

// CreateComplianceClient creates a new Sonobuoy compliance client
func (f *factory) CreateComplianceClient() (*client.SonobuoyClient, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "compliance client failed to load the Kubernetes configuration")
	}
	skc, err := dynamic.NewAPIHelperFromRESTConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "compliance dynamic client failed to be created")
	}
	return client.NewSonobuoyClient(config, skc)
}

// CreateVaultOperatorClient creates a new vault operator client
func (f *factory) CreateVaultOperatorClient() (vaultoperatorclient.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return vaultoperatorclient.NewForConfig(config)
}

// CreateHelm creates a new Helm client
func (f *factory) CreateHelm(verbose bool,
	helmBinary string,
	noTiller bool,
	helmTemplate bool) helm.Helmer {

	if helmBinary == "" {
		helmBinary = "helm"
	}
	featureFlag := "none"
	if helmTemplate {
		featureFlag = "template-mode"
	} else if noTiller {
		featureFlag = "no-tiller-server"
	}
	if verbose {
		log.Logger().Debugf("Using helmBinary %s with feature flag: %s", util.ColorInfo(helmBinary), util.ColorInfo(featureFlag))
	}
	helmCLI := helm.NewHelmCLI(helmBinary, helm.V2, "", verbose)
	var h helm.Helmer = helmCLI
	if helmTemplate {
		kubeClient, ns, _ := f.CreateKubeClient()
		h = helm.NewHelmTemplate(helmCLI, "", kubeClient, ns)
	} else {
		h = helmCLI
	}
	if noTiller && !helmTemplate {
		h.SetHost(helm.GetTillerAddress())
		if err := helm.StartLocalTillerIfNotRunning(); err != nil {
			return h
		}
	}
	return h
}

// CreateCertManagerClient creates a new Kuberntes client for cert-manager resources
func (f *factory) CreateCertManagerClient() (certmngclient.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return certmngclient.NewForConfig(config)
}
