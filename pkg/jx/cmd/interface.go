package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/vault"

	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/table"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	certmngclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	buildclient "github.com/knative/build/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metricsclient "k8s.io/metrics/pkg/client/clientset_generated/clientset"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Factory is the interface defined for jx interactions via the cli
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/jx/cmd Factory -o mocks/factory.go --generate-matchers
type Factory interface {
	//
	// Constructors
	//

	// WithBearerToken creates a factory from a k8s bearer token
	WithBearerToken(token string) Factory

	// ImpersonateUser creates a factory with an impersonated users
	ImpersonateUser(user string) Factory

	//
	// Configuration services
	//

	// CreateAuthConfigService creates a new authentication configuration service
	CreateAuthConfigService(fileName string) (auth.ConfigService, error)

	// CreateJenkinsAuthConfigService creates a new Jenkins authentication configuration service
	CreateJenkinsAuthConfigService(kubernetes.Interface, string) (auth.ConfigService, error)

	// CreateChartmuseumAuthConfigService creates a new Chartmuseum authentication configuration service
	CreateChartmuseumAuthConfigService() (auth.ConfigService, error)

	// CreateIssueTrackerAuthConfigService creates a new issuer tracker configuration service
	CreateIssueTrackerAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	// CreateChatAuthConfigService creates a new chat configuration service
	CreateChatAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	// CreateAddonAuthConfigService creates a new addon auth configuration service
	CreateAddonAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	//
	// Generic clients
	//

	// CreateJenkinsClient creates a new Jenkins client
	CreateJenkinsClient(kubeClient kubernetes.Interface, ns string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gojenkins.JenkinsClient, error)

	// CreateGitProvider creates a new Git provider
	CreateGitProvider(string, string, auth.ConfigService, string, bool, gits.Gitter, terminal.FileReader, terminal.FileWriter, io.Writer) (gits.GitProvider, error)

	// CreateComplianceClient creates a new compliance client
	CreateComplianceClient() (*client.SonobuoyClient, error)

	// CreateSystemVaultClient creates the system vault client for managing the secreets
	CreateSystemVaultClient() (vault.Client, error)

	// CreateVaultClient returns the vault client for given vault
	CreateVaultClient(name string, namespace string) (vault.Client, error)

	// CreateHelm creates a new helm client
	CreateHelm(verbose bool, helmBinary string, noTiller bool, helmTemplate bool) helm.Helmer

	//
	// Kubernetes clients
	//

	// CreateKubeClient creates a new Kubernetes client
	CreateKubeClient() (kubernetes.Interface, string, error)

	// CreateKubeConfig creates the kuberntes configuration
	CreateKubeConfig() (*rest.Config, error)

	// CreateJXClient creates a new Kubernetes client for Jenkins X CRDs
	CreateJXClient() (versioned.Interface, string, error)

	// CreateApiExtensionsClient creates a new Kubernetes ApiExtensions client
	CreateApiExtensionsClient() (apiextensionsclientset.Interface, error)

	// CreateDynamicClient creates a new Kuberntes Dynamic client
	CreateDynamicClient() (*dynamic.APIHelper, string, error)

	// CreateMetricsClient creates a new Kuberntes metrics client
	CreateMetricsClient() (*metricsclient.Clientset, error)

	// CreateKnativeBuildClient create a new Kubernetes client for Knative resources
	CreateKnativeBuildClient() (buildclient.Interface, string, error)

	// CreateVaultOperatorClient creates a new Kuberntes client for Vault operator resources
	CreateVaultOperatorClient() (vaultoperatorclient.Interface, error)

	// CreateCertManagerClient creates a new Kuberntes client for cert-manager resources
	CreateCertManagerClient() (certmngclient.Interface, error)

	//
	// Other methods
	//

	// CreateTable creates a new table
	CreateTable(out io.Writer) table.Table

	// GetJenkinsURL returns the Jenkins URL
	GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error)

	// SetBatch configures the batch modes
	SetBatch(batch bool)

	// For tests only, assert that no actual network connections are being made.
	SetOffline(offline bool)

	// IsInCluster indicates if the execution takes place within a Kuberntes cluster
	IsInCluster() bool

	// IsInCDPipeline indicates if the execution takes place within a CD pipeline
	IsInCDPipeline() bool

	// AuthMergePipelineSecrets merges the current config with the pipeline secrets provided in k8s secrets
	AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error

	// UseVault indicates if the platform is using a Vault to manage the secrets
	UseVault() bool
}
