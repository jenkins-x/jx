package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/vault"

	"github.com/jenkins-x/jx/pkg/expose"

	"github.com/jenkins-x/jx/pkg/kube/resources"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	vaultoperatorclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/jenkins-x/golang-jenkins"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	jxjenkins "github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/jenkins-x/jx/pkg/util"
	certmngclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	tektonclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	buildclient "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	optionServerName       = "name"
	optionServerURL        = "url"
	optionBatchMode        = "batch-mode"
	optionVerbose          = "verbose"
	optionLogLevel         = "log-level"
	optionNoBrew           = "no-brew"
	optionInstallDeps      = "install-dependencies"
	optionSkipAuthSecMerge = "skip-auth-secrets-merge"
)

// ModifyDevEnvironmentFn a callback to create/update the development Environment
type ModifyDevEnvironmentFn func(callback func(env *jenkinsv1.Environment) error) error

// ModifyEnvironmentFn a callback to create/update an Environment
type ModifyEnvironmentFn func(name string, callback func(env *jenkinsv1.Environment) error) error

// CommonOptions contains common options and helper methods
type CommonOptions struct {
	Prow

	In                     terminal.FileReader
	Out                    terminal.FileWriter
	Err                    io.Writer
	Cmd                    *cobra.Command
	Args                   []string
	BatchMode              bool
	Verbose                bool
	LogLevel               string
	NoBrew                 bool
	InstallDependencies    bool
	SkipAuthSecretsMerge   bool
	ServiceAccount         string
	Username               string
	ExternalJenkinsBaseURL string

	factory                clients.Factory
	kubeClient             kubernetes.Interface
	apiExtensionsClient    apiextensionsclientset.Interface
	currentNamespace       string
	devNamespace           string
	jxClient               versioned.Interface
	knbClient              buildclient.Interface
	tektonClient           tektonclient.Interface
	jenkinsClient          gojenkins.JenkinsClient
	git                    gits.Gitter
	helm                   helm.Helmer
	kuber                  kube.Kuber
	vaultOperatorClient    vaultoperatorclient.Interface
	vaultClient            vault.Client
	systemVaultClient      vault.Client
	resourcesInstaller     resources.Installer
	complianceClient       *client.SonobuoyClient
	certManagerClient      certmngclient.Interface
	modifyDevEnvironmentFn ModifyDevEnvironmentFn
	modifyEnvironmentFn    ModifyEnvironmentFn
	environmentsDir        string
	fakeGitProvider        *gits.FakeProvider
}

type ServerFlags struct {
	ServerName string
	ServerURL  string
}

// IsEmpty returns true if the server flags and server URL are tempry
func (f *ServerFlags) IsEmpty() bool {
	return f.ServerName == "" && f.ServerURL == ""
}

// GetFactory lazily creates a Factory if its not already created
func (o *CommonOptions) GetFactory() clients.Factory {
	if o.factory == nil {
		o.factory = clients.NewFactory()
	}
	return o.factory
}

// SetFactory sets the factory to use
func (o *CommonOptions) SetFactory(f clients.Factory) {
	o.factory = f
}

// CreateTable creates a new Table
func (o *CommonOptions) createTable() table.Table {
	return o.factory.CreateTable(o.Out)
}

// NewCommonOptions a helper method to create a new CommonOptions instance
// pre configured in a specific devNamespace
func NewCommonOptions(devNamespace string, factory clients.Factory) CommonOptions {
	return CommonOptions{
		factory:          factory,
		Out:              os.Stdout,
		Err:              os.Stderr,
		currentNamespace: devNamespace,
		devNamespace:     devNamespace,
	}
}

// NewCommonOptionsWithFactory creates a new CommonOptions instance with the
// given factory
func NewCommonOptionsWithFactory(factory clients.Factory) CommonOptions {
	return CommonOptions{
		factory: factory,
	}
}

// SetDevNamespace configures the current dev namespace
func (o *CommonOptions) SetDevNamespace(ns string) {
	o.devNamespace = ns
	o.currentNamespace = ns
	o.kubeClient = nil

	log.Infof("setting the dev namespace to: %s\n", util.ColorInfo(ns))
}

// Debugf outputs the given text to the console if verbose mode is enabled
func (o *CommonOptions) Debugf(format string, a ...interface{}) {
	if o.Verbose {
		log.Infof(format, a...)
	}
}

// addCommonFlags adds the common flags to the given command
func (options *CommonOptions) addCommonFlags(cmd *cobra.Command) {
	defaultBatchMode := false
	if os.Getenv("JX_BATCH_MODE") == "true" {
		defaultBatchMode = true
	}
	cmd.PersistentFlags().BoolVarP(&options.BatchMode, optionBatchMode, "b", defaultBatchMode, "Runs in batch mode without prompting for user input")
	cmd.PersistentFlags().BoolVarP(&options.Verbose, optionVerbose, "", false, "Enables verbose output")
	cmd.PersistentFlags().StringVarP(&options.LogLevel, optionLogLevel, "", logrus.InfoLevel.String(), "Sets the logging level (panic, fatal, error, warning, info, debug)")
	cmd.PersistentFlags().BoolVarP(&options.NoBrew, optionNoBrew, "", false, "Disables brew package manager on MacOS when installing binary dependencies")
	cmd.PersistentFlags().BoolVarP(&options.InstallDependencies, optionInstallDeps, "", false, "Enables automatic dependencies installation when required")
	cmd.PersistentFlags().BoolVarP(&options.SkipAuthSecretsMerge, optionSkipAuthSecMerge, "", false, "Skips merging the secrets from local files with the secrets from Kubernetes cluster")

	options.Cmd = cmd
}

// ApiExtensionsClient return or creates the api extension client
func (o *CommonOptions) ApiExtensionsClient() (apiextensionsclientset.Interface, error) {
	var err error
	if o.apiExtensionsClient == nil {
		o.apiExtensionsClient, err = o.factory.CreateApiExtensionsClient()
		if err != nil {
			return nil, err
		}
	}
	return o.apiExtensionsClient, nil
}

// KubeClient returns or creates the kube client
func (o *CommonOptions) KubeClient() (kubernetes.Interface, error) {
	if o.kubeClient == nil {
		kubeClient, currentNs, err := o.factory.CreateKubeClient()
		if err != nil {
			return nil, err
		}
		o.kubeClient = kubeClient
		o.currentNamespace = currentNs

	}
	return o.kubeClient, nil
}

// KubeClientAndNamespace returns or creates the kube client and the current namespace
func (o *CommonOptions) KubeClientAndNamespace() (kubernetes.Interface, string, error) {
	client, err := o.KubeClient()
	return client, o.currentNamespace, err
}

// SetKubeClient sets the kube client
func (o *CommonOptions) SetKubeClient(kubeClient kubernetes.Interface) {
	o.kubeClient = kubeClient
}

// KubeClientAndDevNamespace returns a kube client and the development namespace
func (o *CommonOptions) KubeClientAndDevNamespace() (kubernetes.Interface, string, error) {
	kubeClient, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return nil, "", err
	}
	if o.devNamespace == "" {
		o.devNamespace, _, err = kube.GetDevNamespace(kubeClient, curNs)
	}
	return kubeClient, o.devNamespace, err
}

// JXClient returns or creates the jx client and current namespace
func (o *CommonOptions) JXClient() (versioned.Interface, string, error) {
	if o.factory == nil {
		return nil, "", errors.New("command factory is not initialized")
	}
	if o.jxClient == nil {
		jxClient, ns, err := o.factory.CreateJXClient()
		if err != nil {
			return nil, ns, err
		}
		o.jxClient = jxClient
		if o.currentNamespace == "" {
			o.currentNamespace = ns
		}
	}
	return o.jxClient, o.currentNamespace, nil
}

// TektonClient lazily creates a new Knative Pipeline client
func (o *CommonOptions) TektonClient() (tektonclient.Interface, string, error) {
	if o.factory == nil {
		return nil, "", errors.New("command factory is not initialized")
	}
	if o.tektonClient == nil {
		tektonClient, ns, err := o.factory.CreateTektonClient()
		if err != nil {
			return nil, ns, err
		}
		o.tektonClient = tektonClient
		if o.currentNamespace == "" {
			o.currentNamespace = ns
		}
	}
	return o.tektonClient, o.currentNamespace, nil
}

// KnativeBuildClient returns or creates the knative build client
func (o *CommonOptions) KnativeBuildClient() (buildclient.Interface, string, error) {
	if o.factory == nil {
		return nil, "", errors.New("command factory is not initialized")
	}
	if o.knbClient == nil {
		knbClient, ns, err := o.factory.CreateKnativeBuildClient()
		if err != nil {
			return nil, ns, err
		}
		o.knbClient = knbClient
		if o.currentNamespace == "" {
			o.currentNamespace = ns
		}
	}
	return o.knbClient, o.currentNamespace, nil
}

// JXClientAndAdminNamespace returns or creates the jx client and admin namespace
func (o *CommonOptions) JXClientAndAdminNamespace() (versioned.Interface, string, error) {
	kubeClient, _, err := o.KubeClientAndNamespace()
	if err != nil {
		return nil, "", err
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, "", err
	}

	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	return jxClient, ns, err
}

// JXClientAndDevNamespace returns and creates the jx client and dev namespace
func (o *CommonOptions) JXClientAndDevNamespace() (versioned.Interface, string, error) {
	if o.jxClient == nil {
		jxClient, ns, err := o.JXClient()
		if err != nil {
			return nil, ns, err
		}
		o.jxClient = jxClient
		if o.currentNamespace == "" {
			o.currentNamespace = ns
		}
	}
	if o.devNamespace == "" {
		client, ns, err := o.KubeClientAndNamespace()
		if err != nil {
			return nil, "", err
		}
		devNs, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return nil, "", err
		}
		if devNs == "" {
			devNs = ns
		}
		o.devNamespace = devNs
	}
	return o.jxClient, o.devNamespace, nil
}

// Git returns the git client
func (o *CommonOptions) Git() gits.Gitter {
	if o.git == nil {
		o.git = gits.NewGitCLI()
	}
	return o.git
}

// SetGit sets the git client
func (o *CommonOptions) SetGit(git gits.Gitter) {
	o.git = git
}

// NewHelm cerates a new helm client from the given list of parameters
func (o *CommonOptions) NewHelm(verbose bool, helmBinary string, noTiller bool, helmTemplate bool) helm.Helmer {
	o.helm = o.factory.CreateHelm(o.Verbose, helmBinary, noTiller, helmTemplate)
	return o.helm
}

// Helm returns or creates the helm client
func (o *CommonOptions) Helm() helm.Helmer {
	if o.helm == nil {
		helmBinary, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			log.Warnf("Failed to retrieve team settings: %v - falling back to default settings...\n", err)
		}
		return o.NewHelm(o.Verbose, helmBinary, noTiller, helmTemplate)
	}
	return o.helm
}

// SetHelm sets the helmer used for this object
func (o *CommonOptions) SetHelm(helmer helm.Helmer) {
	o.helm = helmer
}

// Kube returns the k8s config client
func (o *CommonOptions) Kube() kube.Kuber {
	if o.kuber == nil {
		o.kuber = kube.NewKubeConfig()
	}
	return o.kuber
}

// SetKube  sets the kube config client
func (o *CommonOptions) SetKube(kuber kube.Kuber) {
	o.kuber = kuber
}

// SetResourcesInstaller configures the installer for Kubernetes resources
func (o *CommonOptions) SetResourcesInstaller(installer resources.Installer) {
	o.resourcesInstaller = installer
}

// ResourcesInstaller returns the installer for Kubernetes resources
func (o *CommonOptions) ResourcesInstaller() resources.Installer {
	if o.resourcesInstaller == nil {
		o.resourcesInstaller = resources.NewKubeCtlInstaller("", true, true)
	}
	return o.resourcesInstaller
}

// TeamAndEnvironmentNames returns team and environment namespace
func (o *CommonOptions) TeamAndEnvironmentNames() (string, string, error) {
	kubeClient, currentNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", "", err
	}
	return kube.GetDevNamespace(kubeClient, currentNs)
}

func (o *ServerFlags) addGitServerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.ServerName, optionServerName, "n", "", "The name of the Git server to add a user")
	cmd.Flags().StringVarP(&o.ServerURL, optionServerURL, "u", "", "The URL of the Git server to add a user")
}

// findGitServer finds the Git server from the given flags or returns an error
func (o *CommonOptions) findGitServer(config *auth.AuthConfig, serverFlags *ServerFlags) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, "git", "Try creating one via: jx create git server", false)
}

// findIssueTrackerServer finds the issue tracker server from the given flags or returns an error
func (o *CommonOptions) findIssueTrackerServer(config *auth.AuthConfig, serverFlags *ServerFlags) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, "issues", "Try creating one via: jx create tracker server", false)
}

// findChatServer finds the chat server from the given flags or returns an error
func (o *CommonOptions) findChatServer(config *auth.AuthConfig, serverFlags *ServerFlags) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, "chat", "Try creating one via: jx create chat server", false)
}

// findAddonServer finds the addon server from the given flags or returns an error
func (o *CommonOptions) findAddonServer(config *auth.AuthConfig, serverFlags *ServerFlags, kind string) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, kind, "Try creating one via: jx create addon", true)
}

func (o *CommonOptions) findServer(config *auth.AuthConfig, serverFlags *ServerFlags, defaultKind string, missingServerDescription string, lazyCreate bool) (*auth.AuthServer, error) {
	kind := defaultKind
	var server *auth.AuthServer
	if serverFlags.ServerURL != "" {
		server = config.GetServer(serverFlags.ServerURL)
		if server == nil {
			if lazyCreate {
				return config.GetOrCreateServerName(serverFlags.ServerURL, serverFlags.ServerName, kind), nil
			}
			return nil, util.InvalidOption(optionServerURL, serverFlags.ServerURL, config.GetServerURLs())
		}
	}
	if server == nil && serverFlags.ServerName != "" {
		name := serverFlags.ServerName
		if lazyCreate {
			server = config.GetOrCreateServerName(serverFlags.ServerURL, name, kind)
		} else {
			server = config.GetServerByName(name)
		}
		if server == nil {
			return nil, util.InvalidOption(optionServerName, name, config.GetServerNames())
		}
	}
	if server == nil {
		name := config.CurrentServer
		if name != "" && o.BatchMode {
			server = config.GetServerByName(name)
			if server == nil {
				log.Warnf("Current server %s no longer exists\n", name)
			}
		}
	}
	if server == nil && len(config.Servers) == 1 {
		server = config.Servers[0]
	}
	if server == nil && len(config.Servers) > 1 {
		if o.BatchMode {
			return nil, fmt.Errorf("Multiple servers found. Please specify one via the %s option", optionServerName)
		}
		defaultServerName := ""
		if config.CurrentServer != "" {
			s := config.GetServer(config.CurrentServer)
			if s != nil {
				defaultServerName = s.Name
			}
		}
		name, err := util.PickNameWithDefault(config.GetServerNames(), "Pick server to use: ", defaultServerName, "", o.In, o.Out, o.Err)
		if err != nil {
			return nil, err
		}
		server = config.GetServerByName(name)
		if server == nil {
			return nil, fmt.Errorf("Could not find the server for name %s", name)
		}
	}
	if server == nil {
		return nil, fmt.Errorf("Could not find a %s. %s", kind, missingServerDescription)
	}
	return server, nil
}

func (o *CommonOptions) findService(name string) (string, error) {
	client, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", err
	}
	devNs, _, err := kube.GetDevNamespace(client, ns)
	if err != nil {
		return "", err
	}
	url, err := services.FindServiceURL(client, ns, name)
	if url == "" {
		url, err = services.FindServiceURL(client, devNs, name)
	}
	if url == "" {
		names, err := services.GetServiceNames(client, ns, name)
		if err != nil {
			return "", err
		}
		if len(names) > 1 {
			name, err = util.PickName(names, "Pick service to open: ", "", o.In, o.Out, o.Err)
			if err != nil {
				return "", err
			}
			if name != "" {
				url, err = services.FindServiceURL(client, ns, name)
			}
		} else if len(names) == 1 {
			// must have been a filter
			url, err = services.FindServiceURL(client, ns, names[0])
		}
		if url == "" {
			return "", fmt.Errorf("Could not find URL for service %s in namespace %s", name, ns)
		}
	}
	return url, nil
}

func (o *CommonOptions) findEnvironmentNamespace(envName string) (string, error) {
	client, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", err
	}
	jxClient, _, err := o.JXClient()
	if err != nil {
		return "", err
	}

	devNs, _, err := kube.GetDevNamespace(client, ns)
	if err != nil {
		return "", err
	}

	envMap, envNames, err := kube.GetEnvironments(jxClient, devNs)
	if err != nil {
		return "", err
	}
	env := envMap[envName]
	if env == nil {
		return "", util.InvalidOption(optionEnvironment, envName, envNames)
	}
	answer := env.Spec.Namespace
	if answer == "" {
		return "", fmt.Errorf("Environment %s does not have a Namespace!", envName)
	}
	return answer, nil
}

func (o *CommonOptions) findServiceInNamespace(name string, ns string) (string, error) {
	client, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", err
	}
	if ns == "" {
		ns = curNs
	}
	url, err := services.FindServiceURL(client, ns, name)
	if url == "" {
		names, err := services.GetServiceNames(client, ns, name)
		if err != nil {
			return "", err
		}
		if len(names) > 1 {
			name, err = util.PickName(names, "Pick service to open: ", "", o.In, o.Out, o.Err)
			if err != nil {
				return "", err
			}
			if name != "" {
				url, err = services.FindServiceURL(client, ns, name)
			}
		} else if len(names) == 1 {
			// must have been a filter
			url, err = services.FindServiceURL(client, ns, names[0])
		}
		if url == "" {
			return "", fmt.Errorf("Could not find URL for service %s in namespace %s", name, ns)
		}
	}
	return url, nil
}

func (o *CommonOptions) retry(attempts int, sleep time.Duration, call func() error) (err error) {
	for i := 0; ; i++ {
		err = call()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Warnf("\nretrying after error:%s\n\n", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

// FatalError is a wrapper struct around regular error indicating that re(try) processing flow should be interrupted
// immediately.
type FatalError struct {
	E error
}

func (err *FatalError) Error() string {
	return fmt.Sprintf("fatal error: %s", err.E.Error())
}

func (o *CommonOptions) retryUntilFatalError(attempts int, sleep time.Duration, call func() (*FatalError, error)) (err error) {
	for i := 0; ; i++ {
		fatalErr, err := call()
		if fatalErr != nil {
			return fatalErr.E
		}
		if err == nil {
			return nil
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Infof("retrying after error:%s\n", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (o *CommonOptions) retryQuiet(attempts int, sleep time.Duration, call func() error) (err error) {
	lastMessage := ""
	dot := false

	for i := 0; ; i++ {
		err = call()
		if err == nil {
			if dot {
				log.Blank()
			}
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		message := fmt.Sprintf("retrying after error: %s", err)
		if lastMessage == message {
			log.Info(".")
			dot = true
		} else {
			lastMessage = message
			if dot {
				dot = false
				log.Blank()
			}
			log.Warnf("%s\n\n", lastMessage)
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (o *CommonOptions) retryQuietlyUntilTimeout(timeout time.Duration, sleep time.Duration, call func() error) (err error) {
	timeoutTime := time.Now().Add(timeout)

	lastMessage := ""
	dot := false

	for i := 0; ; i++ {
		err = call()
		if err == nil {
			if dot {
				log.Blank()
			}
			return
		}

		if time.Now().After(timeoutTime) {
			return fmt.Errorf("Timed out after %s, last error: %s", timeout.String(), err)
		}

		time.Sleep(sleep)

		message := fmt.Sprintf("retrying after error: %s", err)
		if lastMessage == message {
			log.Info(".")
			dot = true
		} else {
			lastMessage = message
			if dot {
				dot = false
				log.Blank()
			}
			log.Warnf("%s\n\n", lastMessage)
		}
	}
}

// retryUntilTrueOrTimeout waits until complete is true, an error occurs or the timeout
func (o *CommonOptions) retryUntilTrueOrTimeout(timeout time.Duration, sleep time.Duration, call func() (bool, error)) (err error) {
	timeoutTime := time.Now().Add(timeout)

	for i := 0; ; i++ {
		complete, err := call()
		if complete || err != nil {
			return err
		}
		if time.Now().After(timeoutTime) {
			return fmt.Errorf("Timed out after %s, last error: %s", timeout.String(), err)
		}

		time.Sleep(sleep)
	}
}

func (o *CommonOptions) getJobMap(jenkinsSelector *JenkinsSelectorOptions, filter string) (map[string]gojenkins.Job, error) {
	jobMap := map[string]gojenkins.Job{}
	jenkins, err := o.CreateCustomJenkinsClient(jenkinsSelector)
	if err != nil {
		return jobMap, err
	}
	jobs, err := jenkins.GetJobs()
	if err != nil {
		return jobMap, err
	}
	o.addJobs(jenkins, &jobMap, filter, "", jobs)
	return jobMap, nil
}

func (o *CommonOptions) addJobs(jenkins gojenkins.JenkinsClient, jobMap *map[string]gojenkins.Job, filter string, prefix string, jobs []gojenkins.Job) {
	for _, j := range jobs {
		name := jxjenkins.JobName(prefix, &j)
		if jxjenkins.IsPipeline(&j) {
			if filter == "" || strings.Contains(name, filter) {
				(*jobMap)[name] = j
				continue
			}
		}
		if j.Jobs != nil {
			o.addJobs(jenkins, jobMap, filter, name, j.Jobs)
		} else {
			job, err := jenkins.GetJob(name)
			if err == nil && job.Jobs != nil {
				o.addJobs(jenkins, jobMap, filter, name, job.Jobs)
			}
		}
	}
}
func (o *CommonOptions) tailBuild(jenkinsSelector *JenkinsSelectorOptions, jobName string, build *gojenkins.Build) error {
	jenkins, err := o.CreateCustomJenkinsClient(jenkinsSelector)
	if err != nil {
		return nil
	}

	u, err := url.Parse(build.Url)
	if err != nil {
		return err
	}
	buildPath := u.Path
	log.Infof("%s %s\n", "tailing the log of", fmt.Sprintf("%s #%d", jobName, build.Number))
	// TODO Logger
	return jenkins.TailLog(buildPath, o.Out, time.Second, time.Hour*100)
}

func (o *CommonOptions) pickRemoteURL(config *gitcfg.Config) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	urls := []string{}
	if config.Remotes != nil {
		for _, r := range config.Remotes {
			if r.URLs != nil {
				for _, u := range r.URLs {
					urls = append(urls, u)
				}
			}
		}
	}
	if len(urls) == 1 {
		return urls[0], nil
	}
	url := ""
	if len(urls) > 1 {
		prompt := &survey.Select{
			Message: "Choose a remote git URL:",
			Options: urls,
		}
		err := survey.AskOne(prompt, &url, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}
	return url, nil
}

// todo switch to using expose as a jx plugin
// get existing config from the devNamespace and run expose in the target environment
func (o *CommonOptions) expose(devNamespace, targetNamespace, password string) error {
	certClient, err := o.factory.CreateCertManagerClient()
	if err != nil {
		return errors.Wrap(err, "creating cert-manager client")
	}
	versionsDir, err := o.cloneJXVersionsRepo("")
	if err != nil {
		return errors.Wrapf(err, "failed to clone the Jenkins X versions repository")
	}
	return expose.Expose(o.kubeClient, certClient, devNamespace, targetNamespace, password, o.Helm(), defaultInstallTimeout, versionsDir)
}

func (o *CommonOptions) runExposecontroller(devNamespace, targetNamespace string, ic kube.IngressConfig, services ...string) error {
	versionsDir, err := o.cloneJXVersionsRepo("")
	if err != nil {
		return errors.Wrapf(err, "failed to clone the Jenkins X versions repository")
	}
	return expose.RunExposecontroller(devNamespace, targetNamespace, ic, o.kubeClient, o.Helm(),
		defaultInstallTimeout, versionsDir, services...)
}

// CleanExposecontrollerReources cleans expose controller resources
func (o *CommonOptions) CleanExposecontrollerReources(ns string) {
	expose.CleanExposecontrollerReources(o.kubeClient, ns)
}

func (o *CommonOptions) getDefaultAdminPassword(devNamespace string) (string, error) {
	client, err := o.KubeClient() // cache may not have been created yet...
	if err != nil {
		return "", fmt.Errorf("cannot obtain k8s client %v", err)
	}
	basicAuth, err := client.CoreV1().Secrets(devNamespace).Get(JXInstallConfig, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot find secret %s in namespace %s: %v", kube.SecretBasicAuth, devNamespace, err)
	}
	adminSecrets := basicAuth.Data[AdminSecretsFile]
	adminConfig := config.AdminSecretsConfig{}

	err = yaml.Unmarshal(adminSecrets, &adminConfig)
	if err != nil {
		return "", err
	}
	return adminConfig.Jenkins.JenkinsSecret.Password, nil
}

func (o *CommonOptions) ensureAddonServiceAvailable(serviceName string) (string, error) {
	present, err := services.IsServicePresent(o.kubeClient, serviceName, o.currentNamespace)
	if err != nil {
		return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
	}
	if present {
		url, err := services.GetServiceURLFromName(o.kubeClient, serviceName, o.currentNamespace)
		if err != nil {
			return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
		}
		return url, nil
	}

	// todo ask if user wants to install addon?
	return "", nil
}

func (o *CommonOptions) getJobName() string {
	owner := os.Getenv("REPO_OWNER")
	repo := os.Getenv("REPO_NAME")
	branch := o.getBranchName("")

	if owner != "" && repo != "" && branch != "" {
		return fmt.Sprintf("%s/%s/%s", owner, repo, branch)
	}

	job := os.Getenv("JOB_NAME")
	if job != "" {
		return job
	}
	return ""
}

func (o *CommonOptions) getBranchName(dir string) string {
	branch := builds.GetBranchName()
	if branch == "" {
		if dir == "" {
			dir = "."
		}
		var err error
		branch, err = o.Git().Branch(dir)
		if err != nil {
			log.Warnf("failed to get the git branch name in dir %s\n", dir)
		}
	}
	return branch
}

func (o *CommonOptions) getBuildNumber() string {
	return builds.GetBuildNumber()
}

// VaultOperatorClient returns or creates the vault operator client
func (o *CommonOptions) VaultOperatorClient() (vaultoperatorclient.Interface, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	if o.vaultOperatorClient == nil {
		vaultOperatorClient, err := o.factory.CreateVaultOperatorClient()
		if err != nil {
			return nil, err
		}
		o.vaultOperatorClient = vaultOperatorClient
	}
	return o.vaultOperatorClient, nil
}

// SystemVaultClient return or creates the system vault client
func (o *CommonOptions) SystemVaultClient(namespace string) (vault.Client, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	if o.systemVaultClient == nil {
		systemVaultClient, err := o.factory.CreateSystemVaultClient(namespace)
		if err != nil {
			return nil, err
		}
		o.systemVaultClient = systemVaultClient
	}
	return o.systemVaultClient, nil
}

// VaultClient returns or creates the vault client
func (o *CommonOptions) VaultClient(name string, namespace string) (vault.Client, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	if o.systemVaultClient == nil {
		vaultClient, err := o.factory.CreateVaultClient(name, namespace)
		if err != nil {
			return nil, err
		}
		o.vaultClient = vaultClient
	}
	return o.vaultClient, nil
}

// GetSecretsLocation returns the location of the secrets
func (o *CommonOptions) GetSecretsLocation() secrets.SecretsLocationKind {
	if o.factory == nil {
		return secrets.FileSystemLocationKind
	}
	return o.factory.SecretsLocation()
}

// SetSecretsLocation sets the secrets location
func (o *CommonOptions) SetSecretsLocation(location secrets.SecretsLocationKind, persist bool) error {
	if o.factory == nil {
		return errors.New("command factory is not initialized")
	}
	return o.factory.SetSecretsLocation(location, persist)
}

// ResetSecretsLocation resets the secrets location
func (o *CommonOptions) ResetSecretsLocation() error {
	if o.factory == nil {
		return errors.New("command factory is not initialized")
	}
	o.factory.ResetSecretsLocation()
	return nil
}

// GetWebHookEndpoint returns the webhook endpoint
func (o *CommonOptions) GetWebHookEndpoint() (string, error) {
	_, _, err := o.JXClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to get jxclient")
	}

	_, err = o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to get kube client")
	}

	isProwEnabled, err := o.isProw()
	if err != nil {
		return "", err
	}

	ns, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return "", err
	}

	var webHookUrl string

	if isProwEnabled {
		baseURL, err := services.GetServiceURLFromName(o.kubeClient, "hook", ns)
		if err != nil {
			return "", err
		}

		webHookUrl = util.UrlJoin(baseURL, "hook")
	} else {
		baseURL, err := services.GetServiceURLFromName(o.kubeClient, "jenkins", ns)
		if err != nil {
			return "", err
		}

		webHookUrl = util.UrlJoin(baseURL, "github-webhook/")
	}

	return webHookUrl, nil
}

//ChangeNamespace switches the current jx/K8S namespace to the one specified.
//This is analogous to running `jx namespace cheese`.
func (o *CommonOptions) ChangeNamespace(ns string) {
	nsOptions := &NamespaceOptions{
		CommonOptions: o,
	}
	nsOptions.BatchMode = true
	nsOptions.Args = []string{ns}
	err := nsOptions.Run()
	if err != nil {
		log.Warnf("Failed to set context to namespace %s: %s", ns, err)
	}

	//Reset all the cached clients & namespace values when switching so that they can be properly recalculated for
	//the new namespace.
	o.kubeClient = nil
	o.jxClient = nil
	o.currentNamespace = ""
	o.devNamespace = ""
}

func (o *CommonOptions) GetIn() terminal.FileReader {
	return o.In
}

func (o *CommonOptions) GetOut() terminal.FileWriter {
	return o.Out
}

func (o *CommonOptions) GetErr() io.Writer {
	return o.Err
}

// EnvironmentsDir is the local directory the environments are stored in  - can be faked out for tests
func (o *CommonOptions) EnvironmentsDir() (string, error) {
	if o.environmentsDir == "" {
		var err error
		o.environmentsDir, err = util.EnvironmentsDir()
		if err != nil {
			return "", err
		}
	}
	return o.environmentsDir, nil
}

// SeeAlsoText returns text to describe which other commands to look at which are related to the current command
func SeeAlsoText(commands ...string) string {
	if len(commands) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nSee Also:\n\n")

	for _, command := range commands {
		u := "https://jenkins-x.io/commands/" + strings.Replace(command, " ", "_", -1)
		sb.WriteString(fmt.Sprintf("* %s : [%s](%s)\n", command, u, u))
	}
	sb.WriteString("\n")
	return sb.String()
}

// ComplianceClient returns or creates the compliance client
func (o *CommonOptions) ComplianceClient() (*client.SonobuoyClient, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	if o.complianceClient == nil {
		complianceClient, err := o.factory.CreateComplianceClient()
		if err != nil {
			return nil, err
		}
		o.complianceClient = complianceClient
	}
	return o.complianceClient, nil

}

// CertManagerClient returns or creates the cert-manager client
func (o *CommonOptions) CertManagerClient() (certmngclient.Interface, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	if o.certManagerClient == nil {
		certManagerClient, err := o.factory.CreateCertManagerClient()
		if err != nil {
			return nil, err
		}
		o.certManagerClient = certManagerClient
	}
	return o.certManagerClient, nil
}

// InCluster return true if the command execution takes place in k8s cluster
func (o *CommonOptions) InCluster() bool {
	return o.factory.IsInCluster()
}

// InCDPipeline return true if the command execution takes place in the CD pipeline
func (o *CommonOptions) InCDPipeline() bool {
	return o.factory.IsInCDPipeline()
}

// SetBatchMode configures the batch mode
func (o *CommonOptions) SetBatchMode(batchMode bool) {
	o.factory.SetBatch(batchMode)
}

// AddonAuthConfigService creates the addon auth config service
func (o *CommonOptions) AddonAuthConfigService(secrets *corev1.SecretList) (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateAddonAuthConfigService(secrets)
}

// JenkinsAuthConfigService creates the jenkins auth config service
func (o *CommonOptions) JenkinsAuthConfigService(client kubernetes.Interface, namespace string, selector *JenkinsSelectorOptions) (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	prow, err := o.isProw()
	if err != nil {
		return nil, err
	}
	if prow {
		selector.UseCustomJenkins = true
	}
	jenkinsServiceName := ""
	if selector.IsCustom() {
		kubeClient, ns, err := o.KubeClientAndDevNamespace()
		if err != nil {
			return nil, err
		}
		jenkinsServiceName, err = o.PickCustomJenkinsName(selector, kubeClient, ns)
		if err != nil {
			return nil, err
		}
	}
	return o.factory.CreateJenkinsAuthConfigService(client, namespace, jenkinsServiceName)
}

// ChartmuseumAuthConfigService creates the chart museum auth config service
func (o *CommonOptions) ChartmuseumAuthConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateChartmuseumAuthConfigService()
}

// AuthConfigService creates the auth config service for given file
func (o CommonOptions) AuthConfigService(file string) (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateAuthConfigService(file)
}
