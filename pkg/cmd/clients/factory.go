package clients

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/jenkins-x/jx/pkg/config"

	"github.com/jenkins-x/jx/pkg/kube/cluster"

	"github.com/jenkins-x/jx/pkg/builds"

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
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	build "github.com/knative/build/pkg/client/clientset/versioned"
	kserve "github.com/knative/serving/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/jxfactory"
)

type factory struct {
	jxfactory.Factory

	Batch bool

	secretLocation secrets.SecretLocation
	offline        bool
	jxFactory      jxfactory.Factory
}

var _ Factory = (*factory)(nil)

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory() Factory {
	f := &factory{}
	f.jxFactory = jxfactory.NewFactory()
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
	copy.jxFactory = copy.jxFactory.ImpersonateUser(user)
	return &copy
}

// WithBearerToken returns a new factory with bearer token
func (f *factory) WithBearerToken(token string) Factory {
	copy := *f
	copy.jxFactory = copy.jxFactory.WithBearerToken(token)
	return &copy
}

// CreateJenkinsClient creates a new Jenkins client
func (f *factory) CreateJenkinsClient(kubeClient kubernetes.Interface, ns string, handles util.IOFileHandles) (gojenkins.JenkinsClient, error) {
	svc, err := f.CreateJenkinsAuthConfigService(kubeClient, ns, "")
	if err != nil {
		return nil, err
	}
	url, err := f.GetJenkinsURL(kubeClient, ns)
	if err != nil {
		return nil, fmt.Errorf("%s. Try switching to the Development Tools environment via: jx env dev", err)
	}
	return jenkins.GetJenkinsClient(url, f.Batch, svc, handles)
}

// CreateCustomJenkinsClient creates a new Jenkins client for the given custom Jenkins App
func (f *factory) CreateCustomJenkinsClient(kubeClient kubernetes.Interface, ns string, jenkinsServiceName string, handles util.IOFileHandles) (gojenkins.JenkinsClient, error) {
	svc, err := f.CreateJenkinsAuthConfigService(kubeClient, ns, jenkinsServiceName)
	if err != nil {
		return nil, err
	}
	url, err := f.GetCustomJenkinsURL(kubeClient, ns, jenkinsServiceName)
	if err != nil {
		return nil, fmt.Errorf("%s. Try switching to the Development Tools environment via: jx env dev", err)
	}
	return jenkins.GetJenkinsClient(url, f.Batch, svc, handles)
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
	client, ns, err := f.CreateKubeClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to create the kube client")
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

func (f *factory) CreateJenkinsAuthConfigService(c kubernetes.Interface, ns string, jenkinsServiceName string) (auth.ConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(auth.JenkinsAuthConfigFile, ns, false)

	if jenkinsServiceName == "" {
		jenkinsServiceName = kube.SecretJenkins
	}

	if err != nil {
		return authConfigSvc, err
	}
	config, err := authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}

	customJenkins := jenkinsServiceName != kube.SecretJenkins

	if len(config.Servers) == 0 || customJenkins {
		secretName := jenkinsServiceName
		if customJenkins {
			secretName = jenkinsServiceName + "-auth"
		}
		userAuth := auth.UserAuth{}

		s, err := c.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
		if err != nil {
			if !customJenkins {
				return authConfigSvc, err
			}
		}
		if s != nil {
			userAuth.Username = string(s.Data[kube.JenkinsAdminUserField])
			userAuth.ApiToken = string(s.Data[kube.JenkinsAdminApiToken])
			userAuth.BearerToken = string(s.Data[kube.JenkinsBearTokenField])
		}

		if customJenkins {
			s, err = c.CoreV1().Secrets(ns).Get(jenkinsServiceName, metav1.GetOptions{})
			if err == nil {
				if userAuth.Username == "" {
					userAuth.Username = string(s.Data[kube.JenkinsAdminUserField])
				}
				userAuth.Password = string(s.Data[kube.JenkinsAdminPasswordField])
			}
		}

		svcURL, err := services.FindServiceURL(c, ns, jenkinsServiceName)
		if svcURL == "" {
			return authConfigSvc, fmt.Errorf("unable to find external URL of service %s in namespace %s", jenkinsServiceName, ns)
		}

		u, err := url.Parse(svcURL)
		if err != nil {
			return authConfigSvc, err
		}
		if !userAuth.IsInvalid() || (customJenkins && userAuth.Password != "") {
			if len(config.Servers) == 0 {
				config.Servers = []*auth.AuthServer{
					{
						Name:  u.Host,
						URL:   svcURL,
						Users: []*auth.UserAuth{&userAuth},
					},
				}
			} else {
				server := config.GetOrCreateServer(svcURL)
				server.Name = u.Host
				server.Users = []*auth.UserAuth{&userAuth}
			}
			// lets save the file so that if we call LoadConfig() again we still have this defaulted user auth
			err = authConfigSvc.SaveConfig()
			if err != nil {
				return authConfigSvc, err
			}
		}
	}
	return authConfigSvc, err
}

func (f *factory) CreateChartmuseumAuthConfigService(namespace string) (auth.ConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(auth.ChartmuseumAuthConfigFile, namespace, false)
	if err != nil {
		return authConfigSvc, err
	}
	_, err = authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}
	return authConfigSvc, err
}

func (f *factory) CreateIssueTrackerAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(auth.IssuesAuthConfigFile, namespace, false)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindIssue, f.IsInCDPipeline())
	}
	return authConfigSvc, err
}

func (f *factory) CreateChatAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(auth.ChatAuthConfigFile, namespace, false)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindChat, f.IsInCDPipeline())
	}
	return authConfigSvc, err
}

func (f *factory) CreateAddonAuthConfigService(namespace string, secrets *corev1.SecretList) (auth.ConfigService, error) {
	authConfigSvc, err := f.CreateAuthConfigService(auth.AddonAuthConfigFile, namespace, false)
	if err != nil {
		return authConfigSvc, err
	}
	if secrets != nil {
		config, err := authConfigSvc.LoadConfig()
		if err != nil {
			return authConfigSvc, err
		}
		f.AuthMergePipelineSecrets(config, secrets, kube.ValueKindAddon, f.IsInCDPipeline())
	}
	return authConfigSvc, err
}

func (f *factory) AuthMergePipelineSecrets(config *auth.AuthConfig, secrets *corev1.SecretList, kind string, isCDPipeline bool) error {
	log.Logger().Debug("merging pipeline secrets with local secrets")
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
						ghOwner := labels[kube.LabelGithubAppOwner]
						if ghOwner != "" && isCDPipeline {
							server.Users = append(server.Users, &auth.UserAuth{
								Username:       string(username),
								ApiToken:       string(pwd),
								GithubAppOwner: ghOwner,
							})
						} else if len(username) > 0 && isCDPipeline {
							userAuth := config.FindUserAuth(u, string(username))
							if userAuth == nil {
								userAuth = &auth.UserAuth{
									Username: string(username),
									ApiToken: string(pwd),
								}
							} else if len(pwd) > 0 {
								userAuth.ApiToken = string(pwd)
							}
							config.SetUserAuth(u, userAuth)
							config.UpdatePipelineServer(server, userAuth)
						}
					}
				}
			}
		}
	}
	return nil
}

// CreateAuthConfigService creates a new service saving auth config under the provided name. Depending on the factory,
// It will either save the config to the local file-system, or a Vault
func (f *factory) CreateAuthConfigService(configName string, namespace string, useGitCredentialsFile bool) (auth.ConfigService, error) {
	if f.SecretsLocation() == secrets.VaultLocationKind {
		client, _, err := f.CreateKubeClient()
		if err != nil {
			return nil, errors.Wrap(err, "creating the kube client")
		}
		vaultClient, err := f.CreateSystemVaultClient(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "creating the vault client")
		}
		var authService auth.ConfigService
		configMapClient := client.CoreV1().ConfigMaps(namespace)
		if auth.IsConfigMapVaultAuth(configMapClient) {
			authService = auth.NewConfigmapVaultAuthConfigService(configName, configMapClient, vaultClient)
		} else {
			authService = auth.NewVaultAuthConfigService(configName, vaultClient)
		}
		return authService, nil
	}
	return auth.NewFileAuthConfigService(configName, useGitCredentialsFile)
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
	log.Logger().Debugf("getting vault name for namespace %s", namespace)
	kubeClient, _, err := f.CreateKubeClient()
	if err != nil {
		return "", err
	}
	var name string
	if data, err := kube.ReadInstallValues(kubeClient, namespace); err == nil && data != nil {
		name = data[kube.SystemVaultName]
		log.Logger().Debugf("system vault name from config %s", name)
		if name == "" {
			clusterName := data[kube.ClusterName]
			if clusterName != "" {
				name = kubevault.SystemVaultNameForCluster(clusterName)
				log.Logger().Debugf("vault name %s generated from cluster %s", name, clusterName)
			}
		}
	}

	if name == "" {
		name, err = kubevault.SystemVaultName(f.jxFactory.KubeConfig())
		log.Logger().Debugf("Vault name generated: %s", name)
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
	devNamespace, _, err := kube.GetDevNamespace(kubeClient, defaultNamespace)
	if err != nil {
		return nil, errors.Wrapf(err, "getting the dev namespace from current namespace %q",
			defaultNamespace)
	}

	// Use the dev namespace from default namespace if nothing is specified by the user
	if namespace == "" {
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
	// if there's an issue loading a requirements yaml lets just default to automatic
	useIngressURL := false
	requirements, _, _ := config.LoadRequirementsConfig("")
	jxClient, _, err := f.CreateJXClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating the JX client")
	}
	teamSettings, err := kube.GetDevEnvTeamSettings(jxClient, devNamespace)
	if err != nil {
		return nil, errors.Wrapf(err, "getting team settings from namespace %s", devNamespace)
	}
	reqsFromTeamSettings, _ := config.GetRequirementsConfigFromTeamSettings(teamSettings)

	// allows us to override using the default lookup URL for vault and ensure we always use the ingress. Used in CI.
	if requirements.Vault.DisableURLDiscovery || (reqsFromTeamSettings != nil && reqsFromTeamSettings.Vault.DisableURLDiscovery) {
		useIngressURL = true
	} else {
		useIngressURL = !cluster.IsInCluster()
	}
	certmngClient, err := f.CreateCertManagerClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating the cert-manager client")
	}
	// lets lookup certmanager certificate and check if one exists, it's a selfsigned cert so we need to use insecure SSL
	// when creating the vault client
	// NOTE: insecureSSLWebhook should only ever be used with test clusters as it is insecure
	insecureSSLWebhook, err := kube.IsStagingCertificate(certmngClient, namespace)
	if err != nil {
		// if there's an issue assume we don't need insecure webhooks to keep existing secure behavior
		insecureSSLWebhook = false
	}

	vaultClient, err := clientFactory.NewVaultClient(name, namespace, useIngressURL, insecureSSLWebhook)
	return vault.NewVaultClient(vaultClient), err
}

func (f *factory) CreateJXClient() (versioned.Interface, string, error) {
	return f.jxFactory.CreateJXClient()
}

func (f *factory) CreateKnativeBuildClient() (build.Interface, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.jxFactory.KubeConfig().LoadConfig()
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
	kubeConfig, _, err := f.jxFactory.KubeConfig().LoadConfig()
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

func (f *factory) CreateTektonClient() (tektonclient.Interface, string, error) {
	return f.jxFactory.CreateTektonClient()
}

func (f *factory) CreateDynamicClient() (*dynamic.APIHelper, string, error) {
	config, err := f.CreateKubeConfig()
	if err != nil {
		return nil, "", err
	}
	kubeConfig, _, err := f.jxFactory.KubeConfig().LoadConfig()
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
		return nil, "", fmt.Errorf("failed to create Kubernetes Client")
	}
	ns := ""
	config, _, err := f.jxFactory.KubeConfig().LoadConfig()
	if err != nil {
		return client, ns, err
	}
	ns = kube.CurrentNamespace(config)
	// TODO allow namsepace to be specified as a CLI argument!
	return client, ns, nil
}

func (f *factory) CreateGitProvider(gitURL string, message string, authConfigSvc auth.ConfigService, gitKind string, ghOwner string, batchMode bool, gitter gits.Gitter, handles util.IOFileHandles) (gits.GitProvider, error) {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	return gitInfo.CreateProvider(cluster.IsInCluster(), authConfigSvc, gitKind, ghOwner, gitter, batchMode, handles)
}

func (f *factory) CreateKubeConfig() (*rest.Config, error) {
	if f.offline {
		panic("not supposed to be making a network connection")
	}
	return f.jxFactory.CreateKubeConfig()
}

func (f *factory) CreateTable(out io.Writer) table.Table {
	return table.CreateTable(out)
}

// IsInCDPipeline we should only load the git / issue tracker API tokens if the current pod
// is in a pipeline and running as the Jenkins service account
func (f *factory) IsInCDPipeline() bool {
	// TODO should we let RBAC decide if we can see the Secrets in the dev namespace?
	// or we should test if we are in the cluster and get the current ServiceAccount name?
	buildNumber := builds.GetBuildNumber()
	return buildNumber != "" || os.Getenv("PIPELINE_KIND") != ""
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
		log.Logger().Errorf("Error creating vault operator client %s", err)
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
	if noTiller && !helmTemplate && helmBinary != "helm3" {
		h.SetHost(helm.GetTillerAddress())
		helm.StartLocalTillerIfNotRunning()
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
