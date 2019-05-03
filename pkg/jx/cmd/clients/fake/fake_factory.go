package fake

import (
	"fmt"
	"io"
	"os"

	"github.com/jenkins-x/jx/pkg/builds"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	kservefake "github.com/knative/serving/pkg/client/clientset/versioned/fake"
	apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/kubernetes/fake"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/vault"
	certmngclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	build "github.com/knative/build/pkg/client/clientset/versioned"
	buildfake "github.com/knative/build/pkg/client/clientset/versioned/fake"
	kserve "github.com/knative/serving/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type fakefactory struct {
	Batch bool

	namespace       string
	kubeConfig      kube.Kuber
	impersonateUser string
	bearerToken     string
	secretLocation  secrets.SecretLocation
	offline         bool
}

var _ clients.Factory = (*fakefactory)(nil)

// NewFakeFactory creates a fake factory which uses fake k8s clients for testing
func NewFakeFactory() clients.Factory {
	f := &fakefactory{
		namespace: "jx",
	}
	f.kubeConfig = kube.NewKubeConfig()
	return f
}

func (f *fakefactory) SetBatch(batch bool) {
	f.Batch = batch
}

func (f *fakefactory) SetOffline(offline bool) {
	f.offline = offline
}

// ImpersonateUser returns a new factory impersonating the given user
func (f *fakefactory) ImpersonateUser(user string) clients.Factory {
	copy := *f
	copy.impersonateUser = user
	return &copy
}

// WithBearerToken returns a new factory with bearer token
func (f *fakefactory) WithBearerToken(token string) clients.Factory {
	copy := *f
	copy.bearerToken = token
	return &copy
}

// CreateJenkinsClient creates a new Jenkins client
func (f *fakefactory) CreateJenkinsClient(kubeClient kubernetes.Interface, ns string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gojenkins.JenkinsClient, error) {
	return clients.NewFactory().CreateJenkinsClient(kubeClient, ns, in, out, errOut)
}

// CreateCustomJenkinsClient creates a new Jenkins client for the given custom Jenkins App
func (f *fakefactory) CreateCustomJenkinsClient(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gojenkins.JenkinsClient, error) {
	return clients.NewFactory().CreateCustomJenkinsClient(kubeClient, ns, jenkinsServiceName, in, out, errOut)
}

// GetJenkinsURL gets the Jenkins URL for the given namespace
func (f *fakefactory) GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error) {
	return clients.NewFactory().GetJenkinsURL(kubeClient, ns)
}

// GetCustomJenkinsURL gets a custom jenkins App service URL
func (f *fakefactory) GetCustomJenkinsURL(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string) (string, error) {
	return clients.NewFactory().GetCustomJenkinsURL(kubeClient, ns, jenkinsServiceName)
}

// CreateJenkinsAuthConfigService creates a new Jenkins authentication configuration service
func (f *fakefactory) CreateJenkinsAuthConfigService(c kubernetes.Interface, ns string, jenkinsServiceName string) (auth.ConfigService, error) {
	return clients.NewFactory().CreateJenkinsAuthConfigService(c, ns, jenkinsServiceName)
}

// CreateChartmuseumAuthConfigService creates a new Chartmuseum authentication configuration service
func (f *fakefactory) CreateChartmuseumAuthConfigService(namespace string) (auth.ConfigService, error) {
	return clients.NewFactory().CreateChartmuseumAuthConfigService(namespace)
}

// CreateIssueTrackerAuthConfigService creates a new issuer tracker configuration service
func (f *fakefactory) CreateIssueTrackerAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	return clients.NewFactory().CreateIssueTrackerAuthConfigService(namespace, secrets)
}

// CreateChatAuthConfigService creates a new chat configuration service
func (f *fakefactory) CreateChatAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	return clients.NewFactory().CreateChatAuthConfigService(namespace, secrets)
}

// CreateAddonAuthConfigService creates a new addon auth configuration service
func (f *fakefactory) CreateAddonAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	return clients.NewFactory().CreateAddonAuthConfigService(namespace, secrets)
}

// AuthMergePipelineSecrets merges the current config with the pipeline secrets provided in k8s secrets
func (f *fakefactory) AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error {
	return clients.NewFactory().AuthMergePipelineSecrets(config, secrets, kind, isCDPipeline)
}

// CreateAuthConfigService creates a new service saving auth config under the provided name. Depending on the factory,
// It will either save the config to the local file-system, or a Vault
func (f *fakefactory) CreateAuthConfigService(configName string, namespace string) (auth.ConfigService, error) {
	if f.SecretsLocation() == secrets.VaultLocationKind {
		vaultClient, err := f.CreateSystemVaultClient(namespace)
		authService := auth.NewVaultAuthConfigService(configName, vaultClient)
		return authService, err
	}
	return auth.NewFileAuthConfigService(configName)
}

// SecretsLocation indicates the location where the secrets are stored
func (f *fakefactory) SecretsLocation() secrets.SecretsLocationKind {
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
func (f *fakefactory) SetSecretsLocation(location secrets.SecretsLocationKind, persist bool) error {
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
func (f *fakefactory) ResetSecretsLocation() {
	f.secretLocation = nil
}

// CreateSystemVaultClient gets the system vault client for managing the secrets
func (f *fakefactory) CreateSystemVaultClient(namespace string) (vault.Client, error) {
	name, err := f.getVaultName(namespace)
	if err != nil {
		return nil, err
	}
	return f.CreateVaultClient(name, namespace)
}

// getVaultName gets the vault name from install configuration or builds a new name from
// cluster name
func (f *fakefactory) getVaultName(namespace string) (string, error) {
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
func (f *fakefactory) CreateVaultClient(name string, namespace string) (vault.Client, error) {
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

// CreateKubeClient creates a new Kubernetes client
func (f *fakefactory) CreateKubeClient() (kubernetes.Interface, string, error) {
	return fake.NewSimpleClientset(), f.namespace, nil
}

// CreateJXClient creates a new Kubernetes client for Jenkins X CRDs
func (f *fakefactory) CreateJXClient() (versioned.Interface, string, error) {
	return v1fake.NewSimpleClientset(), f.namespace, nil
}

// CreateApiExtensionsClient creates a new Kubernetes ApiExtensions client
func (f *fakefactory) CreateApiExtensionsClient() (apiextensionsclientset.Interface, error) {
	return apifake.NewSimpleClientset(), nil
}

func (f *fakefactory) CreateKnativeBuildClient() (build.Interface, string, error) {
	return buildfake.NewSimpleClientset(), f.namespace, nil
}

// CreateKnativeServeClient create a new Kubernetes client for Knative serve resources
func (f *fakefactory) CreateKnativeServeClient() (kserve.Interface, string, error) {
	return kservefake.NewSimpleClientset(), f.namespace, nil
}

// CreateTektonClient create a new Kubernetes client for Tekton resources
func (f *fakefactory) CreateTektonClient() (tektonclient.Interface, string, error) {
	return tektonfake.NewSimpleClientset(), f.namespace, nil
}

// CreateDynamicClient creates a new Kubernetes Dynamic client
func (f *fakefactory) CreateDynamicClient() (*dynamic.APIHelper, string, error) {
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

// CreateMetricsClient creates a new Kubernetes metrics client
func (f *fakefactory) CreateMetricsClient() (*metricsclient.Clientset, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return metricsclient.NewForConfig(config)
}

// CreateGitProvider creates a new Git provider
func (f *fakefactory) CreateGitProvider(gitURL string, message string, authConfigSvc auth.ConfigService, gitKind string, batchMode bool, gitter gits.Gitter, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gits.GitProvider, error) {
	return clients.NewFactory().CreateGitProvider(gitURL, message, authConfigSvc, gitKind, batchMode, gitter, in, out, errOut)
}

// CreateKubeConfig creates the kubernetes configuration
func (f *fakefactory) CreateKubeConfig() (*rest.Config, error) {
	return clients.NewFactory().CreateKubeConfig()
}

func (f *fakefactory) getImpersonateUser() string {
	user := f.impersonateUser
	if user == "" {
		// this is really only used for testing really
		user = os.Getenv("JX_IMPERSONATE_USER")
	}
	return user
}

// CreateTable creates a new table
func (f *fakefactory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(out)
}

// IsInCDPipeline we should only load the git / issue tracker API tokens if the current pod
// is in a pipeline and running as the Jenkins service account
func (f *fakefactory) IsInCDPipeline() bool {
	// TODO should we let RBAC decide if we can see the Secrets in the dev namespace?
	// or we should test if we are in the cluster and get the current ServiceAccount name?
	buildNumber := builds.GetBuildNumber()
	return buildNumber != "" || os.Getenv("PIPELINE_KIND") != ""
}

// function to tell if we are running incluster
func (f *fakefactory) IsInCluster() bool {
	_, err := rest.InClusterConfig()
	if err != nil {
		return false
	}
	return true
}

// CreateComplianceClient creates a new Sonobuoy compliance client
func (f *fakefactory) CreateComplianceClient() (*client.SonobuoyClient, error) {
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
func (f *fakefactory) CreateVaultOperatorClient() (vaultoperatorclient.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return vaultoperatorclient.NewForConfig(config)
}

// CreateHelm creates a new Helm client
func (f *fakefactory) CreateHelm(verbose bool,
	helmBinary string,
	noTiller bool,
	helmTemplate bool) helm.Helmer {

	return clients.NewFactory().CreateHelm(verbose,
		helmBinary,
		noTiller,
		helmTemplate)
}

// CreateCertManagerClient creates a new Kuberntes client for cert-manager resources
func (f *fakefactory) CreateCertManagerClient() (certmngclient.Interface, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	return certmngclient.NewForConfig(config)
}
