package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"

	"os"

	"time"

	"path/filepath"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
)

const (
	optionServerName = "name"
	optionServerURL  = "url"
)

// CommonOptions contains common options and helper methods
type CommonOptions struct {
	Factory   cmdutil.Factory
	Out       io.Writer
	Err       io.Writer
	Cmd       *cobra.Command
	Args      []string
	BatchMode bool
	Verbose   bool
	Headless  bool

	// common cached clients
	kubeClient       *kubernetes.Clientset
	currentNamespace string
	devNamespace     string
	jxClient         *versioned.Clientset
	jenkinsClient    *gojenkins.Jenkins
}

type ServerFlags struct {
	ServerName string
	ServerURL  string
}

func (f *ServerFlags) IsEmpty() bool {
	return f.ServerName == "" && f.ServerURL == ""
}

func addGitRepoOptionsArguments(cmd *cobra.Command, repositoryOptions *gits.GitRepositoryOptions) {
	cmd.Flags().StringVarP(&repositoryOptions.ServerURL, "git-provider-url", "", "https://github.com", "The git server URL to create new git repositories inside")
	cmd.Flags().StringVarP(&repositoryOptions.Username, "git-username", "", "", "The git username to use for creating new git repositories")
	cmd.Flags().StringVarP(&repositoryOptions.ApiToken, "git-api-token", "", "", "The git API token to use for creating new git repositories")
}

func (c *CommonOptions) CreateTable() table.Table {
	return c.Factory.CreateTable(c.Out)
}

func (c *CommonOptions) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(c.Out, format, a...)
}

func (o *CommonOptions) runCommandFromDir(dir, name string, args ...string) error {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = o.Out
	e.Stderr = o.Err
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) runCommand(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) runCommandQuietly(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	return e.Run()
}

func (o *CommonOptions) runCommandInteractive(interactive bool, name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	if interactive {
		e.Stdin = os.Stdin
	}
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

// getCommandOutput evaluates the given command and returns the trimmed output
func (o *CommonOptions) getCommandOutput(dir string, name string, args ...string) (string, error) {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	return text, err
}

func (options *CommonOptions) addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "", false, "Enable verbose logging")
	cmd.Flags().BoolVarP(&options.Headless, "headless", "", false, "Enable headless operation if using browser automation")
	options.Cmd = cmd
}

func (o *CommonOptions) KubeClient() (*kubernetes.Clientset, string, error) {
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

func (o *CommonOptions) JXClient() (*versioned.Clientset, string, error) {
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

func (o *CommonOptions) JXClientAndDevNamespace() (*versioned.Clientset, string, error) {
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

func (o *CommonOptions) GitServerKind(gitInfo *gits.GitRepositoryInfo) (string, error) {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return "", err
	}

	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return "", err
	}
	err = kube.RegisterGitServiceCRD(apisClient)
	if err != nil {
		return "", err
	}

	return kube.GetGitServiceKind(jxClient, devNs, gitInfo.Host)
}

func (o *CommonOptions) JenkinsClient() (*gojenkins.Jenkins, error) {
	if o.jenkinsClient == nil {
		jenkins, err := o.Factory.CreateJenkinsClient()
		if err != nil {
			return nil, err
		}
		o.jenkinsClient = jenkins
	}
	return o.jenkinsClient, nil
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

// gitProviderForURL returns a GitProvider for the given git URL
func (o *CommonOptions) gitProviderForURL(gitURL string, message string) (gits.GitProvider, error) {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigServiceForURL(gitInfo.HostURL())
	if err != nil {
		return nil, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return nil, err
	}
	return gitInfo.PickOrCreateProvider(authConfigSvc, message, o.BatchMode, gitKind)
}

func (o *ServerFlags) addGitServerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.ServerName, optionServerName, "n", "", "The name of the git server to add a user")
	cmd.Flags().StringVarP(&o.ServerURL, optionServerURL, "u", "", "The URL of the git server to add a user")
}

// findGitServer finds the git server from the given flags or returns an error
func (o *CommonOptions) findGitServer(config *auth.AuthConfig, serverFlags *ServerFlags) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, "git server", "Try creating one via: jx create git server")
}

// findGitServer finds the issue tracker server from the given flags or returns an error
func (o *CommonOptions) findIssueTrackerServer(config *auth.AuthConfig, serverFlags *ServerFlags) (*auth.AuthServer, error) {
	return o.findServer(config, serverFlags, "issue tracker server", "Try creating one via: jx create tracker server")
}

func (o *CommonOptions) findServer(config *auth.AuthConfig, serverFlags *ServerFlags, kind string, missingServerDescription string) (*auth.AuthServer, error) {
	var server *auth.AuthServer
	if serverFlags.ServerURL != "" {
		server = config.GetServer(serverFlags.ServerURL)
		if server == nil {
			return nil, util.InvalidOption(optionServerURL, serverFlags.ServerURL, config.GetServerURLs())
		}
	}
	if server == nil && serverFlags.ServerName != "" {
		server = config.GetServerByName(serverFlags.ServerName)
		if server == nil {
			return nil, util.InvalidOption(optionServerName, serverFlags.ServerName, config.GetServerNames())
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
	f := o.Factory
	client, ns, err := f.CreateClient()
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
	f := o.Factory
	client, ns, err := f.CreateClient()
	if err != nil {
		return "", err
	}
	jxClient, _, err := f.CreateJXClient()
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
	f := o.Factory
	client, curNs, err := f.CreateClient()
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

func (o *CommonOptions) registerLocalHelmRepo(repoName, ns string) error {
	if repoName == "" {
		repoName = kube.LocalHelmRepoName
	}
	// TODO we should use the auth package to keep a list of server login/pwds
	// TODO we have a chartmuseumAuth.yaml now but sure yet if that's the best thing to do
	username := "admin"
	password := "admin"

	// lets check if we have a local helm repository
	client, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	u, err := kube.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return err
	}
	u2, err := url.Parse(u)
	if err != nil {
		return err
	}
	if u2.User == nil {
		u2.User = url.UserPassword(username, password)
	}
	helmUrl := u2.String()
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	text, err := o.getCommandOutput("", "helm", "repo", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(text, "\n")
	remove := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			fields := strings.Fields(t)
			if len(fields) > 1 {
				if fields[0] == repoName {
					if fields[1] == helmUrl {
						return nil
					} else {
						remove = true
					}
				}
			}
		}
	}
	if remove {
		err = o.runCommand("helm", "repo", "remove", repoName)
		if err != nil {
			return err
		}
	}
	return o.runCommand("helm", "repo", "add", repoName, helmUrl)
}

// installChart installs the given chart
func (o *CommonOptions) installChart(releaseName string, chart string, version string, ns string, helmUpdate bool) error {
	if helmUpdate {
		err := o.runCommand("helm", "repo", "update")
		if err != nil {
			return err
		}
	}
	timeout := fmt.Sprintf("--timeout=%s", defaultInstallTimeout)
	args := []string{"upgrade", "--install", timeout}
	if version != "" {
		args = append(args, "--version", version)
	}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	args = append(args, releaseName, chart)
	return o.runCommand("helm", args...)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	args := []string{"delete"}
	if purge {
		args = append(args, "--purge")
	}
	args = append(args, releaseName)
	return o.runCommand("helm", args...)
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

func (o *CommonOptions) getJobMap(filter string) (map[string]*gojenkins.Job, error) {
	jobMap := map[string]*gojenkins.Job{}
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

func (o *CommonOptions) addJobs(jobMap *map[string]*gojenkins.Job, filter string, prefix string, jobs []gojenkins.Job) {
	jenkins, err := o.JenkinsClient()
	if err != nil {
		return
	}
	for _, j := range jobs {
		name := jobName(prefix, &j)

		if IsPipeline(&j) {
			if filter == "" || strings.Contains(name, filter) {
				(*jobMap)[name] = &j
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

func (*CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// lets try find the chart file
	chartFile := filepath.Join(dir, "Chart.yaml")
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return "", err
	}
	if !exists {
		// lets try find all the chart files
		files, err := filepath.Glob("*/Chart.yaml")
		if err != nil {
			return "", err
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob("*/*/Chart.yaml")
			if err != nil {
				return "", err
			}
			if len(files) > 0 {
				chartFile = files[0]
				return chartFile, nil
			}
		}
	}
	return "", nil
}

func (o *CommonOptions) discoverGitURL(gitConf string) (string, error) {
	if gitConf == "" {
		return "", fmt.Errorf("No GitConfDir defined!")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return "", fmt.Errorf("Failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	url := gits.GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = gits.GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.pickRemoteURL(cfg)
			if err != nil {
				return "", err
			}
		}
	}
	return url, nil
}
