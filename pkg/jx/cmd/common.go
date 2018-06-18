package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	core_v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	optionServerName        = "name"
	optionServerURL         = "url"
	exposecontrollerVersion = "2.3.56"
	exposecontroller        = "exposecontroller"
	exposecontrollerChart   = "jenkins-x/exposecontroller"
)

// CommonOptions contains common options and helper methods
type CommonOptions struct {
	Factory        cmdutil.Factory
	Out            io.Writer
	Err            io.Writer
	Cmd            *cobra.Command
	Args           []string
	BatchMode      bool
	Verbose        bool
	Headless       bool
	NoBrew         bool
	ServiceAccount string

	// common cached clients
	kubeClient          kubernetes.Interface
	apiExtensionsClient apiextensionsclientset.Interface
	currentNamespace    string
	devNamespace        string
	jxClient            versioned.Interface
	jenkinsClient       *gojenkins.Jenkins
}

type ServerFlags struct {
	ServerName string
	ServerURL  string
}

func (f *ServerFlags) IsEmpty() bool {
	return f.ServerName == "" && f.ServerURL == ""
}

func (c *CommonOptions) Stdout() io.Writer {
	if c.Out != nil {
		return c.Out
	}
	return os.Stdout
}

func (c *CommonOptions) CreateTable() table.Table {
	return c.Factory.CreateTable(c.Stdout())
}

// Printf outputs the given text to the console
func (c *CommonOptions) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(c.Stdout(), format, a...)
}

// Debugf outputs the given text to the console if verbose mode is enabled
func (c *CommonOptions) Debugf(format string, a ...interface{}) (n int, err error) {
	if c.Verbose {
		return fmt.Fprintf(c.Stdout(), format, a...)
	}
	return 0, nil
}

func (options *CommonOptions) addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "", false, "Enable verbose logging")
	cmd.Flags().BoolVarP(&options.Headless, "headless", "", false, "Enable headless operation if using browser automation")
	cmd.Flags().BoolVarP(&options.NoBrew, "no-brew", "", false, "Disables the use of brew on MacOS to install or upgrade command line dependencies")
	options.Cmd = cmd
}

func (o *CommonOptions) CreateApiExtensionsClient() (apiextensionsclientset.Interface, error) {
	var err error
	if o.apiExtensionsClient == nil {
		o.apiExtensionsClient, err = o.Factory.CreateApiExtensionsClient()
		if err != nil {
			return nil, err
		}
	}
	return o.apiExtensionsClient, nil
}

func (o *CommonOptions) KubeClient() (kubernetes.Interface, string, error) {
	if o.kubeClient == nil {
		kubeClient, currentNs, err := o.Factory.CreateClient()
		if err != nil {
			return nil, "", err
		}
		o.kubeClient = kubeClient
		o.currentNamespace = currentNs
	}
	return o.kubeClient, o.currentNamespace, nil
}

func (o *CommonOptions) JXClient() (versioned.Interface, string, error) {
	if o.jxClient == nil {
		jxClient, ns, err := o.Factory.CreateJXClient()
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
		client, ns, err := o.KubeClient()
		if err != nil {
			return nil, "", err
		}
		devNs, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return nil, "", err
		}
		o.devNamespace = devNs
	}
	return o.jxClient, o.devNamespace, nil
}

func (o *CommonOptions) JenkinsClient() (*gojenkins.Jenkins, error) {
	if o.jenkinsClient == nil {
		kubeClient, ns, err := o.KubeClient()
		if err != nil {
			return nil, err
		}

		jenkins, err := o.Factory.CreateJenkinsClient(kubeClient, ns)
		if err != nil {
			return nil, err
		}
		o.jenkinsClient = jenkins
	}
	return o.jenkinsClient, nil
}
func (o *CommonOptions) GetJenkinsURL() (string, error) {
	kubeClient, ns, err := o.KubeClient()
	if err != nil {
		return "", err
	}

	return o.Factory.GetJenkinsURL(kubeClient, ns)
}

func (o *CommonOptions) TeamAndEnvironmentNames() (string, string, error) {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return "", "", err
	}
	return kube.GetDevNamespace(kubeClient, currentNs)
}

// warnf generates a warning
func (o *CommonOptions) warnf(format string, a ...interface{}) {
	o.Printf(util.ColorWarning("WARNING: "+format), a...)
}

func (o *ServerFlags) addGitServerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.ServerName, optionServerName, "n", "", "The name of the git server to add a user")
	cmd.Flags().StringVarP(&o.ServerURL, optionServerURL, "u", "", "The URL of the git server to add a user")
}

// findGitServer finds the git server from the given flags or returns an error
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
				o.warnf("Current server %s no longer exists\n", name)
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
		name, err := util.PickNameWithDefault(config.GetServerNames(), "Pick server to use: ", defaultServerName)
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
	client, ns, err := o.KubeClient()
	if err != nil {
		return "", err
	}
	devNs, _, err := kube.GetDevNamespace(client, ns)
	if err != nil {
		return "", err
	}
	url, err := kube.FindServiceURL(client, ns, name)
	if url == "" {
		url, err = kube.FindServiceURL(client, devNs, name)
	}
	if url == "" {
		names, err := kube.GetServiceNames(client, ns, name)
		if err != nil {
			return "", err
		}
		if len(names) > 1 {
			name, err = util.PickName(names, "Pick service to open: ")
			if err != nil {
				return "", err
			}
			if name != "" {
				url, err = kube.FindServiceURL(client, ns, name)
			}
		} else if len(names) == 1 {
			// must have been a filter
			url, err = kube.FindServiceURL(client, ns, names[0])
		}
		if url == "" {
			return "", fmt.Errorf("Could not find URL for service %s in namespace %s", name, ns)
		}
	}
	return url, nil
}

func (o *CommonOptions) findEnvironmentNamespace(envName string) (string, error) {
	client, ns, err := o.KubeClient()
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
	client, curNs, err := o.KubeClient()
	if err != nil {
		return "", err
	}
	if ns == "" {
		ns = curNs
	}
	url, err := kube.FindServiceURL(client, ns, name)
	if url == "" {
		names, err := kube.GetServiceNames(client, ns, name)
		if err != nil {
			return "", err
		}
		if len(names) > 1 {
			name, err = util.PickName(names, "Pick service to open: ")
			if err != nil {
				return "", err
			}
			if name != "" {
				url, err = kube.FindServiceURL(client, ns, name)
			}
		} else if len(names) == 1 {
			// must have been a filter
			url, err = kube.FindServiceURL(client, ns, names[0])
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

		o.Printf("retrying after error:%s\n", err)
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
				o.Printf("\n")
			}
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		message := fmt.Sprintf("retrying after error: %s", err)
		if lastMessage == message {
			o.Printf(".")
			dot = true
		} else {
			lastMessage = message
			if dot {
				dot = false
				o.Printf("\n")
			}
			o.Printf("%s\n", lastMessage)
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
				o.Printf("\n")
			}
			return
		}

		if time.Now().After(timeoutTime) {
			return fmt.Errorf("Timed out after %s, last error: %s", timeout.String(), err)
		}

		time.Sleep(sleep)

		message := fmt.Sprintf("retrying after error: %s", err)
		if lastMessage == message {
			o.Printf(".")
			dot = true
		} else {
			lastMessage = message
			if dot {
				dot = false
				o.Printf("\n")
			}
			o.Printf("%s\n", lastMessage)
		}
	}
}

func (o *CommonOptions) getJobMap(filter string) (map[string]gojenkins.Job, error) {
	jobMap := map[string]gojenkins.Job{}
	jenkins, err := o.JenkinsClient()
	if err != nil {
		return jobMap, err
	}
	jobs, err := jenkins.GetJobs()
	if err != nil {
		return jobMap, err
	}
	o.addJobs(&jobMap, filter, "", jobs)
	return jobMap, nil
}

func (o *CommonOptions) addJobs(jobMap *map[string]gojenkins.Job, filter string, prefix string, jobs []gojenkins.Job) {
	jenkins, err := o.JenkinsClient()
	if err != nil {
		return
	}

	for _, j := range jobs {
		name := jobName(prefix, &j)
		if IsPipeline(&j) {
			if filter == "" || strings.Contains(name, filter) {
				(*jobMap)[name] = j
				continue
			}
		}
		if j.Jobs != nil {
			o.addJobs(jobMap, filter, name, j.Jobs)
		} else {
			job, err := jenkins.GetJob(name)
			if err == nil && job.Jobs != nil {
				o.addJobs(jobMap, filter, name, job.Jobs)
			}
		}
	}
}
func (o *CommonOptions) tailBuild(jobName string, build *gojenkins.Build) error {
	jenkins, err := o.JenkinsClient()
	if err != nil {
		return nil
	}

	u, err := url.Parse(build.Url)
	if err != nil {
		return err
	}
	buildPath := u.Path
	o.Printf("%s %s\n", util.ColorStatus("tailing the log of"), util.ColorInfo(fmt.Sprintf("%s #%d", jobName, build.Number)))
	return jenkins.TailLog(buildPath, o.Out, time.Second, time.Hour*100)
}

func (o *CommonOptions) pickRemoteURL(config *gitcfg.Config) (string, error) {
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
		err := survey.AskOne(prompt, &url, nil)
		if err != nil {
			return "", err
		}
	}
	return url, nil
}

// todo switch to using exposecontroller as a jx plugin
// get existing config from the devNamespace and run exposecontroller in the target environment
func (o *CommonOptions) expose(devNamespace, targetNamespace, releaseName, password string) error {

	_, err := o.kubeClient.CoreV1().Secrets(targetNamespace).Get(kube.SecretBasicAuth, v1.GetOptions{})
	if err != nil {
		data := make(map[string][]byte)

		if password != "" {
			hash := config.HashSha(password)
			data[kube.AUTH] = []byte(fmt.Sprintf("admin:{SHA}%s", hash))
		} else {
			basicAuth, err := o.kubeClient.CoreV1().Secrets(devNamespace).Get(kube.SecretBasicAuth, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("cannot find secret %s in namespace %s: %v", kube.SecretBasicAuth, devNamespace, err)
			}
			data = basicAuth.Data
		}

		sec := &core_v1.Secret{
			Data: data,
			ObjectMeta: v1.ObjectMeta{
				Name: kube.SecretBasicAuth,
			},
		}
		_, err := o.kubeClient.CoreV1().Secrets(targetNamespace).Create(sec)
		if err != nil {
			return fmt.Errorf("cannot create secret %s in target namespace %s: %v", kube.SecretBasicAuth, targetNamespace, err)
		}
	}

	exposecontrollerConfig, err := kube.GetTeamExposecontrollerConfig(o.kubeClient, devNamespace)
	if err != nil {
		return fmt.Errorf("cannot get existing team exposecontroller config from namespace %s: %v", devNamespace, err)
	}

	var exValues []string
	if targetNamespace != devNamespace {
		// run exposecontroller using existing team config
		exValues = []string{
			"config.exposer=" + exposecontrollerConfig["exposer"],
			"config.domain=" + exposecontrollerConfig["domain"],
			"config.http=" + exposecontrollerConfig["http"],
			"config.tls-acme=" + exposecontrollerConfig["tls-acme"],
		}
	}

	err = o.installChart("expose"+releaseName, exposecontrollerChart, exposecontrollerVersion, targetNamespace, true, exValues)
	if err != nil {
		return fmt.Errorf("exposecontroller deployment failed: %v", err)
	}
	err = kube.WaitForJobToSucceeded(o.kubeClient, targetNamespace, exposecontroller, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed waiting for exposecontroller job to succeed: %v", err)
	}
	return kube.DeleteJob(o.kubeClient, targetNamespace, exposecontroller)

}

func (o *CommonOptions) getDefaultAdminPassword(devNamespace string) (string, error) {
	basicAuth, err := o.kubeClient.CoreV1().Secrets(devNamespace).Get(JXInstallConfig, v1.GetOptions{})
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
	present, err := kube.IsServicePresent(o.kubeClient, serviceName, o.currentNamespace)
	if err != nil {
		return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
	}
	if present {
		url, err := kube.GetServiceURLFromName(o.kubeClient, serviceName, o.currentNamespace)
		if err != nil {
			return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
		}
		return url, nil
	}

	// todo ask if user wants to install addon?
	return "", nil
}
