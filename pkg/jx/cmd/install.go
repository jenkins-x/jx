package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/addon"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/src-d/go-git.v4"
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type InstallOptions struct {
	CommonOptions
	gits.GitRepositoryOptions
	CreateJenkinsUserOptions
	CreateEnvOptions
	config.AdminSecretsService

	InitOptions InitOptions
	Flags       InstallFlags
}

type InstallFlags struct {
	Domain                   string
	Provider                 string
	CloudEnvRepository       string
	LocalHelmRepoName        string
	Namespace                string
	NoDefaultEnvironments    bool
	HelmTLS                  bool
	DefaultEnvironmentPrefix string
	LocalCloudEnvironment    bool
	Timeout                  string
	RegisterLocalHelmRepo    bool
	CleanupTempFiles         bool
	EnvironmentGitOwner      string
}

type Secrets struct {
	Login string
	Token string
}

const (
	JX_GIT_TOKEN                   = "JX_GIT_TOKEN"
	JX_GIT_USER                    = "JX_GIT_USER"
	DEFAULT_CLOUD_ENVIRONMENTS_URL = "https://github.com/jenkins-x/cloud-environments"

	GitSecretsFile        = "gitSecrets.yaml"
	AdminSecretsFile      = "adminSecrets.yaml"
	ExtraValuesFile       = "extraValues.yaml"
	JXInstallConfig       = "jx-install-config"
	defaultInstallTimeout = "6000"
)

var (
	instalLong = templates.LongDesc(`
		Installs the Jenkins X platform on a Kubernetes cluster

		Requires a --git-username and --git-api-token that can be used to create a new token.
		This is so the Jenkins X platform can git tag your releases

		For more documentation see: [http://jenkins-x.io/getting-started/install-on-cluster/](http://jenkins-x.io/getting-started/install-on-cluster/)

		The current requirements are:
		* RBAC is enabled on the cluster
		* insecure docker registry is enabled for docker registries running locally inside kubernetes on the service IP range. See the above documentation for more detail 

`)

	instalExample = templates.Examples(`
		# Default installer which uses interactive prompts to generate git secrets
		jx install

		# Install with a GitHub personal access token
		jx install --git-username jenkins-x-bot --git-api-token 9fdbd2d070cd81eb12bca87861bcd850

		# If you know the cloud provider you can pass this as a CLI argument. E.g. for AWS
		jx install --provider=aws
`)
)

// NewCmdGet creates a command object for the generic "install" action, which
// installs the jenkins-x platform on a kubernetes cluster.
func NewCmdInstall(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {

	options := createInstallOptions(f, out, errOut)

	cmd := &cobra.Command{
		Use:     "install [flags]",
		Short:   "Install Jenkins X in the current Kubernetes cluster",
		Long:    instalLong,
		Example: instalExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	options.addCommonFlags(cmd)
	options.addInstallFlags(cmd, false)

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the kubernetes cluster.  Supported providers: "+KubernetesProviderOptions())
	return cmd
}

func createInstallOptions(f cmdutil.Factory, out io.Writer, errOut io.Writer) InstallOptions {
	commonOptions := CommonOptions{
		Factory: f,
		Out:     out,
		Err:     errOut,
	}
	options := InstallOptions{
		CreateJenkinsUserOptions: CreateJenkinsUserOptions{
			Username: "admin",
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory:  f,
					Out:      out,
					Err:      errOut,
					Headless: true,
				},
			},
		},
		GitRepositoryOptions: gits.GitRepositoryOptions{},
		CommonOptions:        commonOptions,
		CreateEnvOptions: CreateEnvOptions{
			HelmValuesConfig: config.HelmValuesConfig{
				ExposeController: &config.ExposeController{},
			},
			Options: v1.Environment{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.EnvironmentSpec{
					PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
				},
			},
			PromotionStrategy:      string(v1.PromotionStrategyTypeAutomatic),
			ForkEnvironmentGitRepo: kube.DefaultEnvironmentGitRepoURL,
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory:   f,
					Out:       out,
					Err:       errOut,
					Headless:  true,
					BatchMode: true,
				},
			},
		},
		InitOptions: InitOptions{
			CommonOptions: commonOptions,
			Flags:         InitFlags{},
		},
		AdminSecretsService: config.AdminSecretsService{},
	}
	return options
}

func (options *InstallOptions) addInstallFlags(cmd *cobra.Command, includesInit bool) {
	flags := &options.Flags
	cmd.Flags().StringVarP(&flags.CloudEnvRepository, "cloud-environment-repo", "", DEFAULT_CLOUD_ENVIRONMENTS_URL, "Cloud Environments git repo")
	cmd.Flags().StringVarP(&flags.LocalHelmRepoName, "local-helm-repo-name", "", kube.LocalHelmRepoName, "The name of the helm repository for the installed Chart Museum")
	cmd.Flags().BoolVarP(&flags.NoDefaultEnvironments, "no-default-environments", "", false, "Disables the creation of the default Staging and Production environments")
	cmd.Flags().StringVarP(&flags.DefaultEnvironmentPrefix, "default-environment-prefix", "", "", "Default environment repo prefix, your git repos will be of the form 'environment-$prefix-$envName'")
	cmd.Flags().BoolVarP(&flags.LocalCloudEnvironment, "local-cloud-environment", "", false, "Ignores default cloud-environment-repo and uses current directory ")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "", "jx", "The namespace the Jenkins X platform should be installed into")
	cmd.Flags().StringVarP(&flags.Timeout, "timeout", "", defaultInstallTimeout, "The number of seconds to wait for the helm install to complete")
	cmd.Flags().StringVarP(&flags.EnvironmentGitOwner, "environment-git-owner", "", "", "The git provider organisation to create the environment git repositories in")
	cmd.Flags().BoolVarP(&flags.RegisterLocalHelmRepo, "register-local-helmrepo", "", false, "Registers the Jenkins X chartmuseum registry with your helm client [default false]")
	cmd.Flags().BoolVarP(&flags.CleanupTempFiles, "cleanup-temp-files", "", true, "Cleans up any temporary values.yaml used by helm install [default true]")
	cmd.Flags().BoolVarP(&flags.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")

	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)
	options.HelmValuesConfig.AddExposeControllerValues(cmd, true)
	options.AdminSecretsService.AddAdminSecretsValues(cmd)
	options.InitOptions.addInitFlags(cmd)
}

// Run implements this command
func (options *InstallOptions) Run() error {
	client, _, err := options.Factory.CreateClient()
	if err != nil {
		return err
	}
	options.kubeClient = client

	context, err := options.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	ns := options.Flags.Namespace
	if ns == "" {
		f := options.Factory
		_, ns, _ = f.CreateClient()
		if err != nil {
			return err
		}
	}

	err = kube.EnsureNamespaceCreated(client, ns, map[string]string{kube.LabelTeam: ns}, nil)
	if err != nil {
		return fmt.Errorf("Failed to ensure the namespace %s is created: %s\nIs this an RBAC issue on your cluster?", ns, err)
	}

	err = options.runCommand("kubectl", "config", "set-context", context, "--namespace", ns)
	if err != nil {
		return err
	}

	options.Flags.Provider, err = options.GetCloudProvider(options.Flags.Provider)
	if err != nil {
		return err
	}

	err = options.installRequirements(options.Flags.Provider)
	if err != nil {
		return err
	}

	if options.Flags.Provider == AKS {
		/**
		 * create a cluster admin role
		 */
		err = options.createClusterAdmin()
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "AlreadyExists") {
				log.Success("role cluster-admin already exists for the cluster")

			} else {
				return err
			}
		} else {
			log.Success("created role cluster-admin")
		}
	}

	initOpts := &options.InitOptions
	initOpts.Flags.Provider = options.Flags.Provider
	initOpts.Flags.Namespace = options.Flags.Namespace
	initOpts.BatchMode = options.BatchMode

	currentContext, err := options.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	if currentContext == "minikube" {
		if options.Flags.Provider == "" {
			options.Flags.Provider = MINIKUBE
		}
		ip, err := options.getCommandOutput("", "minikube", "ip")
		if err != nil {
			return err
		}
		options.Flags.Domain = ip + ".nip.io"
	}

	if initOpts.Flags.Domain == "" && options.Flags.Domain != "" {
		initOpts.Flags.Domain = options.Flags.Domain
	}

	// lets default the helm domain
	exposeController := options.CreateEnvOptions.HelmValuesConfig.ExposeController
	if exposeController != nil && exposeController.Config.Domain == "" && options.Flags.Domain != "" {
		exposeController.Config.Domain = options.Flags.Domain
		log.Success("set exposeController Config Domain " + exposeController.Config.Domain + "\n")
	}
	if exposeController != nil {
		if isOpenShiftProvider(options.Flags.Provider) {
			exposeController.Config.Exposer = "Route"
		}
	}

	err = initOpts.Run()
	if err != nil {
		return err
	}

	if isOpenShiftProvider(options.Flags.Provider) {
		err = options.enableOpenShiftSCC(ns)
		if err != nil {
			return err
		}
	}

	// share the init domain option with the install options
	if initOpts.Flags.Domain != "" && options.Flags.Domain == "" {
		options.Flags.Domain = initOpts.Flags.Domain
	}

	// get secrets to use in helm install
	secrets, err := options.getGitSecrets()
	if err != nil {
		return err
	}

	err = options.AdminSecretsService.NewAdminSecretsConfig()
	if err != nil {
		return err
	}

	adminSecrets, err := options.AdminSecretsService.Secrets.String()
	if err != nil {
		return err
	}

	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	if helmConfig.ExposeController.Config.Domain == "" {
		helmConfig.ExposeController.Config.Domain = options.InitOptions.Flags.Domain
	}
	domain := helmConfig.ExposeController.Config.Domain
	if domain != "" && addon.IsAddonEnabled("gitea") {
		helmConfig.Jenkins.Servers.GetOrCreateFirstGitea().Url = "http://gitea-gitea." + ns + "." + domain
	}

	// lets add any GitHub Enterprise servers
	gitAuthCfg, err := options.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	err = options.addGitServersToJenkinsConfig(helmConfig, gitAuthCfg)
	if err != nil {
		return err
	}

	config, err := helmConfig.String()
	if err != nil {
		return err
	}

	// clone the environments repo
	wrkDir := filepath.Join(util.HomeDir(), ".jx", "cloud-environments")
	err = options.cloneJXCloudEnvironmentsRepo(wrkDir)
	if err != nil {
		return err
	}

	// run  helm install setting the token and domain values
	if options.Flags.Provider == "" {
		return fmt.Errorf("No kubernetes provider found to match cloud-environment with")
	}
	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(options.Flags.Provider)))
	if _, err := os.Stat(wrkDir); os.IsNotExist(err) {
		return fmt.Errorf("cloud environment dir %s not found", makefileDir)
	}

	// create a temporary file that's used to pass current git creds to helm in order to create a secret for pipelines to tag releases
	dir, err := util.ConfigDir()
	if err != nil {
		return err
	}

	secretsFileName := filepath.Join(dir, GitSecretsFile)
	err = ioutil.WriteFile(secretsFileName, []byte(secrets), 0644)
	if err != nil {
		return err
	}

	adminSecretsFileName := filepath.Join(dir, AdminSecretsFile)
	err = ioutil.WriteFile(adminSecretsFileName, []byte(adminSecrets), 0644)
	if err != nil {
		return err
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return err
	}

	data := make(map[string][]byte)
	data[ExtraValuesFile] = []byte(config)
	data[AdminSecretsFile] = []byte(adminSecrets)
	data[GitSecretsFile] = []byte(secrets)

	jxSecrets := &core_v1.Secret{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: JXInstallConfig,
		},
	}
	_, err = options.kubeClient.CoreV1().Secrets(ns).Create(jxSecrets)
	if err != nil {
		return err
	}

	options.Printf("Generated helm values %s\n", util.ColorInfo(configFileName))

	timeout := options.Flags.Timeout
	if timeout == "" {
		timeout = defaultInstallTimeout
	}
	arg := fmt.Sprintf("ARGS=--values=%s --values=%s --values=%s --namespace=%s --timeout=%s", secretsFileName, adminSecretsFileName, configFileName, ns, timeout)

	// run the helm install
	options.Printf("Installing Jenkins X platform helm chart from: %s\n", makefileDir)

	options.Verbose = true
	err = options.runCommandFromDir(makefileDir, "make", arg, "install")
	if err != nil {
		return err
	}

	if options.Flags.CleanupTempFiles {
		err = os.Remove(secretsFileName)
		if err != nil {
			return err
		}

		err = os.Remove(configFileName)
		if err != nil {
			return err
		}
	}

	err = options.waitForInstallToBeReady(ns)
	if err != nil {
		return err
	}
	log.Infof("Jenkins X deployments ready in namespace %s\n", ns)

	addonConfig, err := addon.LoadAddonsConfig()
	if err != nil {
		return err
	}

	for _, ac := range addonConfig.Addons {
		if ac.Enabled {
			err = options.installAddon(ac.Name)
			if err != nil {
				return fmt.Errorf("Failed to install addon %s: %s", ac.Name, err)
			}
		}
	}

	options.logAdminPassword()

	if !options.Flags.NoDefaultEnvironments {
		log.Info("Getting Jenkins API Token\n")
		err = options.retry(3, 2*time.Second, func() (err error) {
			options.CreateJenkinsUserOptions.Password = options.AdminSecretsService.Flags.DefaultAdminPassword
			options.CreateJenkinsUserOptions.UseBrowser = true
			if options.BatchMode {
				options.CreateJenkinsUserOptions.BatchMode = true
				options.CreateJenkinsUserOptions.Headless = true
				options.Printf("Attempting to find the Jenkins API Token with the browser in headless mode...")
			}
			err = options.CreateJenkinsUserOptions.Run()
			return
		})
		if err != nil {
			return err
		}

		if options.Flags.DefaultEnvironmentPrefix == "" {
			options.Flags.DefaultEnvironmentPrefix = strings.ToLower(randomdata.SillyName())
		}

		log.Info("Creating default staging and production environments\n")
		options.CreateEnvOptions.Options.Name = "staging"
		options.CreateEnvOptions.Options.Spec.Label = "Staging"
		options.CreateEnvOptions.Options.Spec.Order = 100
		options.CreateEnvOptions.GitRepositoryOptions.Owner = options.Flags.EnvironmentGitOwner
		options.CreateEnvOptions.Prefix = options.Flags.DefaultEnvironmentPrefix
		if options.BatchMode {
			options.CreateEnvOptions.BatchMode = options.BatchMode
		}

		err = options.CreateEnvOptions.Run()
		if err != nil {
			return err
		}
		options.CreateEnvOptions.Options.Name = "production"
		options.CreateEnvOptions.Options.Spec.Label = "Production"
		options.CreateEnvOptions.Options.Spec.Order = 200
		options.CreateEnvOptions.Options.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual
		options.CreateEnvOptions.PromotionStrategy = string(v1.PromotionStrategyTypeManual)
		options.CreateEnvOptions.GitRepositoryOptions.Owner = options.Flags.EnvironmentGitOwner
		if options.BatchMode {
			options.CreateEnvOptions.BatchMode = options.BatchMode
		}

		err = options.CreateEnvOptions.Run()
		if err != nil {
			return err
		}
	}

	err = options.saveChartmuseumAuthConfig()
	if err != nil {
		return err
	}

	if options.Flags.RegisterLocalHelmRepo {
		err = options.registerLocalHelmRepo(options.Flags.LocalHelmRepoName, ns)
		if err != nil {
			return err
		}
	}

	log.Success("\nJenkins X installation completed successfully\n")

	options.logAdminPassword()

	options.Printf("\nTo import existing projects into Jenkins:     %s\n", util.ColorInfo("jx import"))
	options.Printf("To create a new Spring Boot microservice:       %s\n", util.ColorInfo("jx create spring -d web -d actuator"))
	options.Printf("To create a new microservice from a quickstart: %s\n", util.ColorInfo("jx create quickstart"))
	return nil
}

func isOpenShiftProvider(provider string) bool {
	switch provider {
	case OPENSHIFT, MINISHIFT:
		return true
	default:
		return false
	}
}

func (o *InstallOptions) enableOpenShiftSCC(ns string) error {
	o.Printf("Enabling anyui for the Jenkins service account in namespace %s\n", ns)
	err := o.runCommand("oc", "adm", "policy", "add-scc-to-user", "anyuid", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	err = o.runCommand("oc", "adm", "policy", "add-scc-to-user", "hostaccess", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	err = o.runCommand("oc", "adm", "policy", "add-scc-to-user", "privileged", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	// try fix monocular
	return o.runCommand("oc", "adm", "policy", "add-scc-to-user", "anyuid", "system:serviceaccount:"+ns+":default")
}

func (options *InstallOptions) logAdminPassword() {
	astrix := `
	
	********************************************************
	
	     NOTE: %s
	
	********************************************************
	
	`
	options.Printf(astrix, fmt.Sprintf("Your admin password is: %s", util.ColorInfo(options.AdminSecretsService.Flags.DefaultAdminPassword)))
}

// clones the jenkins-x cloud-environments repo to a local working dir
func (o *InstallOptions) cloneJXCloudEnvironmentsRepo(wrkDir string) error {
	log.Infof("Cloning the Jenkins X cloud environments repo to %s\n", wrkDir)
	if o.Flags.CloudEnvRepository == "" {
		return fmt.Errorf("No cloud environment git URL")
	}
	if o.Flags.LocalCloudEnvironment {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current working directory %v", err)
		}
		log.Infof("Copying local dir to %s\n", wrkDir)

		return util.CopyDir(currentDir, wrkDir, true)
	}
	_, err := git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           o.Flags.CloudEnvRepository,
		ReferenceName: "refs/heads/master",
		SingleBranch:  true,
		Progress:      o.Out,
	})
	if err != nil {
		if strings.Contains(err.Error(), "repository already exists") {
			flag := false
			if o.BatchMode {
				flag = true
			} else {
				confirm := &survey.Confirm{
					Message: "A local Jenkins X cloud environments repository already exists, recreate with latest?",
					Default: true,
				}
				err := survey.AskOne(confirm, &flag, nil)
				if err != nil {
					return err
				}
			}
			if flag {
				err := os.RemoveAll(wrkDir)
				if err != nil {
					return err
				}

				return o.cloneJXCloudEnvironmentsRepo(wrkDir)
			}
		} else {
			return err
		}
	}
	return nil
}

// returns secrets that are used as values during the helm install
func (o *InstallOptions) getGitSecrets() (string, error) {
	username, token, err := o.getGitToken()
	if err != nil {
		return "", err
	}

	server := o.GitRepositoryOptions.ServerURL
	if server == "" {
		return "", fmt.Errorf("No Git Server found")
	}
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")

	url := fmt.Sprintf("%s:%s@%s", username, token, server)
	// TODO convert to a struct
	pipelineSecrets := `
PipelineSecrets:
  GitCreds: |-
    https://%s
    http://%s`
	return fmt.Sprintf(pipelineSecrets, url, url), nil
}

// returns the Git Token that should be used by Jenkins X to setup credentials to clone repos and creates a secret for pipelines to tag a release
func (o *InstallOptions) getGitToken() (string, string, error) {
	username := o.GitRepositoryOptions.Username
	if username == "" {
		if os.Getenv(JX_GIT_USER) != "" {
			username = os.Getenv(JX_GIT_USER)
		}
	}
	if username != "" {
		// first check git-token flag
		if o.GitRepositoryOptions.ApiToken != "" {
			return username, o.GitRepositoryOptions.ApiToken, nil
		}

		// second check for an environment variable
		if os.Getenv(JX_GIT_TOKEN) != "" {
			return username, os.Getenv(JX_GIT_TOKEN), nil
		}
	}
	o.Printf("Lets set up a git username and API token to be able to perform CI/CD\n\n")
	userAuth, err := o.getGitUser("")
	if err != nil {
		return "", "", err
	}
	return userAuth.Username, userAuth.ApiToken, nil
}

func (o *InstallOptions) waitForInstallToBeReady(ns string) error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	log.Warnf("waiting for install to be ready, if this is the first time then it will take a while to download images")

	return kube.WaitForAllDeploymentsToBeReady(client, ns, 30*time.Minute)

}

func (options *InstallOptions) saveChartmuseumAuthConfig() error {

	authConfigSvc, err := options.Factory.CreateChartmuseumAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	if options.ServerFlags.IsEmpty() {
		url := ""
		url, err = options.findService(kube.ServiceChartMuseum)
		if err != nil {
			return err
		}
		server = config.GetOrCreateServer(url)
	} else {
		server, err = options.findServer(config, &options.ServerFlags, "chartmuseum server", "Try installing one via: jx create team", false)
		if err != nil {
			return err
		}
	}

	user := &auth.UserAuth{
		Username: "admin",
		Password: options.AdminSecretsService.Flags.DefaultAdminPassword,
	}

	server.Users = append(server.Users, user)

	config.CurrentServer = server.URL
	return authConfigSvc.SaveConfig()
}

func (o *InstallOptions) getGitUser(message string) (*auth.UserAuth, error) {
	var userAuth *auth.UserAuth
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return userAuth, err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	gitProvider := o.GitRepositoryOptions.ServerURL
	if gitProvider != "" {
		server = config.GetOrCreateServer(gitProvider)
	} else {
		server, err = config.PickServer("Which git provider?", o.BatchMode)
		if err != nil {
			return userAuth, err
		}
		o.GitRepositoryOptions.ServerURL = server.URL
	}
	url := server.URL
	if message == "" {
		message = fmt.Sprintf("%s username for CI/CD pipelines:", server.Label())
	}
	userAuth, err = config.PickServerUserAuth(server, message, o.BatchMode)
	if err != nil {
		return userAuth, err
	}
	if userAuth.IsInvalid() {
		f := func(username string) error {
			gits.PrintCreateRepositoryGenerateAccessToken(server, username, o.Out)
			return nil
		}

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(server.Label(), userAuth, defaultUserName, false, o.BatchMode, f)
		if err != nil {
			return userAuth, err
		}

		// TODO lets verify the auth works

		err = authConfigSvc.SaveUserAuth(url, userAuth)
		if err != nil {
			return userAuth, fmt.Errorf("failed to store git auth configuration %s", err)
		}
		if userAuth.IsInvalid() {
			return userAuth, fmt.Errorf("you did not properly define the user authentication")
		}
	}
	return userAuth, nil
}

func (o *InstallOptions) installAddon(name string) error {
	o.Printf("Installing addon %s\n", util.ColorInfo(name))

	options := &CreateAddonOptions{
		CreateOptions: CreateOptions{
			CommonOptions: o.CommonOptions,
		},
		HelmUpdate: true,
	}
	if name == "gitea" {
		options.ReleaseName = defaultGiteaReleaseName
		giteaOptions := &CreateAddonGiteaOptions{
			CreateAddonOptions: *options,
			Chart:              kube.ChartGitea,
		}
		return giteaOptions.Run()
	}
	return options.CreateAddon(name)
}

func (o *InstallOptions) addGitServersToJenkinsConfig(helmConfig *config.HelmValuesConfig, gitAuthCfg auth.AuthConfigService) error {
	cfg := gitAuthCfg.Config()
	for _, server := range cfg.Servers {
		if server.Kind == "github" {
			u := server.URL
			if !gits.IsGitHubServerURL(u) {
				sc := config.JenkinsGithubServersValuesConfig{
					Name: server.Name,
					Url:  gits.GitHubEnterpriseApiEndpointURL(u),
				}
				helmConfig.Jenkins.Servers.GHE = append(helmConfig.Jenkins.Servers.GHE, sc)
			}
		}
	}
	return nil
}
