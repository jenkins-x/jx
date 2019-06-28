package clients

import (
	"io"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/vault"

	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/table"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	certmngclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	buildclient "github.com/knative/build/pkg/client/clientset/versioned"
	kserve "github.com/knative/serving/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Factory is the interface defined for jx interactions via the cli
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/cmd/clients Factory -o mocks/factory.go
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

	// CreateConfigService creates a new authentication configuration service
	CreateConfigService(fileName string, serverKind string, serviceKind string) (auth.ConfigService, error)

	// CreateGitConfigService creates a new git config service
	CreateGitConfigService() (auth.ConfigService, error)

	// CreateJenkinsConfigService creates a new Jenkins authentication configuration service
	CreateJenkinsConfigService() (auth.ConfigService, error)

	// CreateChartmuseumConfigService creates a new Chartmuseum authentication configuration service
	CreateChartmuseumConfigService() (auth.ConfigService, error)

	// CreateIssueTrackerConfigService creates a new issuer tracker configuration service
	CreateIssueTrackerConfigService() (auth.ConfigService, error)

	// CreateChatConfigService creates a new chat configuration service
	CreateChatConfigService() (auth.ConfigService, error)

	// CreateAddonConfigService creates a new addon auth configuration service
	CreateAddonConfigService() (auth.ConfigService, error)

	//
	// Generic clients
	//

	// CreateGitProvider creates a new Git provider
	CreateGitProvider(gitURL string, git gits.Gitter) (gits.GitProvider, error)

	// CreateJenkinsClient creates a new Jenkins client
	CreateJenkinsClient(kubeClient kubernetes.Interface, ns string) (gojenkins.JenkinsClient, error)

	// CreateCustomJenkinsClient creates a new Jenkins client for the custom Jenkins App with the jenkinsServiceName
	CreateCustomJenkinsClient(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string) (gojenkins.JenkinsClient, error)

	// CreateComplianceClient creates a new compliance client
	CreateComplianceClient() (*client.SonobuoyClient, error)

	// CreateSystemVaultClient creates the system vault client for managing the secrets
	CreateSystemVaultClient(namespace string) (vault.Client, error)

	// CreateVaultClient returns the vault client for given vault
	CreateVaultClient(name string, namespace string) (vault.Client, error)

	// CreateHelm creates a new helm client
	CreateHelm(verbose bool, helmBinary string, noTiller bool, helmTemplate bool) helm.Helmer

	//
	// Kubernetes clients
	//

	// CreateKubeClient creates a new Kubernetes client
	CreateKubeClient() (kubernetes.Interface, string, error)

	// CreateKubeConfig creates the kubernetes configuration
	CreateKubeConfig() (*rest.Config, error)

	// CreateJXClient creates a new Kubernetes client for Jenkins X CRDs
	CreateJXClient() (versioned.Interface, string, error)

	// CreateApiExtensionsClient creates a new Kubernetes ApiExtensions client
	CreateApiExtensionsClient() (apiextensionsclientset.Interface, error)

	// CreateDynamicClient creates a new Kubernetes Dynamic client
	CreateDynamicClient() (*dynamic.APIHelper, string, error)

	// CreateMetricsClient creates a new Kubernetes metrics client
	CreateMetricsClient() (*metricsclient.Clientset, error)

	// CreateTektonClient create a new Kubernetes client for Tekton resources
	CreateTektonClient() (tektonclient.Interface, string, error)

	// CreateKnativeBuildClient create a new Kubernetes client for Knative Build resources
	CreateKnativeBuildClient() (buildclient.Interface, string, error)

	// CreateKnativeServeClient create a new Kubernetes client for Knative serve resources
	CreateKnativeServeClient() (kserve.Interface, string, error)

	// CreateVaultOperatorClient creates a new Kubernetes client for Vault operator resources
	CreateVaultOperatorClient() (vaultoperatorclient.Interface, error)

	// CreateCertManagerClient creates a new Kubernetes client for cert-manager resources
	CreateCertManagerClient() (certmngclient.Interface, error)

	//
	// Other methods
	//

	// CreateTable creates a new table
	CreateTable(out io.Writer) table.Table

	// GetJenkinsURL returns the Jenkins URL
	GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error)

	// GetCustomJenkinsURL gets a custom jenkins App service URL
	GetCustomJenkinsURL(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string) (string, error)

	// SetBatch configures the batch modes
	SetBatch(batch bool)

	// For tests only, assert that no actual network connections are being made.
	SetOffline(offline bool)

	// IsInCluster indicates if the execution takes place within a Kubernetes cluster
	IsInCluster() bool

	// SecretsLocation inidcates the location of the secrets
	SecretsLocation() secrets.SecretsLocationKind

	// SetSecretsLocation configures the secrets location in memory. It will persist the secrets location in a
	// config map if the persist flag is active.
	SetSecretsLocation(location secrets.SecretsLocationKind, persist bool) error

	// ResetSecretsLocation resets the location of the secrets
	ResetSecretsLocation()
}
