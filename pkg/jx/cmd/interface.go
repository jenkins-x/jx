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

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	buildclient "github.com/knative/build/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metricsclient "k8s.io/metrics/pkg/client/clientset_generated/clientset"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Factory is the interface defined for jx interactions via the cli
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/jx/cmd Factory -o mocks/factory.go --generate-matchers
type Factory interface {
	WithBearerToken(token string) Factory

	ImpersonateUser(user string) Factory

	CreateJenkinsClient(kubeClient kubernetes.Interface, ns string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gojenkins.JenkinsClient, error)

	GetJenkinsURL(kubeClient kubernetes.Interface, ns string) (string, error)

	CreateAuthConfigService(fileName string) (auth.ConfigService, error)

	CreateJenkinsAuthConfigService(kubernetes.Interface, string) (auth.ConfigService, error)

	CreateChartmuseumAuthConfigService() (auth.ConfigService, error)

	CreateIssueTrackerAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	CreateChatAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	CreateAddonAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error)

	CreateKubeClient() (kubernetes.Interface, string, error)

	CreateGitProvider(string, string, auth.ConfigService, string, bool, gits.Gitter, terminal.FileReader, terminal.FileWriter, io.Writer) (gits.GitProvider, error)

	CreateKubeConfig() (*rest.Config, error)

	// CreateJXClienta creates a JXclient for interacting with JX resources.
	CreateJXClient() (versioned.Interface, string, error)

	CreateApiExtensionsClient() (apiextensionsclientset.Interface, error)

	CreateDynamicClient() (*dynamic.APIHelper, string, error)

	CreateMetricsClient() (*metricsclient.Clientset, error)

	CreateComplianceClient() (*client.SonobuoyClient, error)

	CreateKnativeBuildClient() (buildclient.Interface, string, error)

	CreateTable(out io.Writer) table.Table

	SetBatch(batch bool)

	// For tests only, assert that no actual network connections are being made.
	SetOffline(offline bool)

	IsInCluster() bool

	IsInCDPipeline() bool

	AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error

	CreateVaultOperatorClient() (vaultoperatorclient.Interface, error)

	GetHelm(verbose bool, helmBinary string, noTiller bool, helmTemplate bool) helm.Helmer

	// UseVault indicates if the platform is using a Vault to manage the secrets
	UseVault() bool

	// GetSystemVaultClient gets the system vault client for managing the secreets
	GetSystemVaultClient() (vault.Client, error)

	// GetVaultClient returns the vault client for given vault
	GetVaultClient(name string, namespace string) (vault.Client, error)
}
