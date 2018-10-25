package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/addon"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/iks"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"gopkg.in/src-d/go-git.v4"
	core_v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InstallOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
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

// InstallFlags flags for the install command
type InstallFlags struct {
	Domain                   string
	ExposeControllerPathMode string
	DockerRegistry           string
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
	InstallOnly              bool
	EnvironmentGitOwner      string
	Version                  string
	Prow                     bool
	DisableSetKubeContext    bool
}

// Secrets struct for secrets
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
	CloudEnvValuesFile    = "myvalues.yaml"
	CloudEnvSecretsFile   = "secrets.yaml"
	defaultInstallTimeout = "6000"

	ServerlessJenkins   = "Serverless Jenkins"
	StaticMasterJenkins = "Static Master Jenkins"
)

var (
	instalLong = templates.LongDesc(`
		Installs the Jenkins X platform on a Kubernetes cluster

		Requires a --git-username and --git-api-token that can be used to create a new token.
		This is so the Jenkins X platform can git tag your releases

		For more documentation see: [https://jenkins-x.io/getting-started/install-on-cluster/](https://jenkins-x.io/getting-started/install-on-cluster/)

		The current requirements are:

		*RBAC is enabled on the cluster

		*Insecure Docker registry is enabled for Docker registries running locally inside Kubernetes on the service IP range. See the above documentation for more detail

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

// NewCmdInstall creates a command object for the generic "install" action, which
// installs the jenkins-x platform on a Kubernetes cluster.
func NewCmdInstall(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {

	options := CreateInstallOptions(f, in, out, errOut)

	cmd := &cobra.Command{
		Use:     "install [flags]",
		Short:   "Install Jenkins X in the current Kubernetes cluster",
		Long:    instalLong,
		Example: instalExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	options.addCommonFlags(cmd)
	options.addInstallFlags(cmd, false)

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the Kubernetes cluster.  Supported providers: "+KubernetesProviderOptions())
	return cmd
}

// CreateInstallOptions creates the options for jx install
func CreateInstallOptions(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) InstallOptions {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,
		Out:     out,
		Err:     errOut,
	}
	options := InstallOptions{
		CreateJenkinsUserOptions: CreateJenkinsUserOptions{
			Username: "admin",
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory:  f,
					In:       in,
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
					In:        in,
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
	flags.addCloudEnvOptions(cmd)
	cmd.Flags().StringVarP(&flags.LocalHelmRepoName, "local-helm-repo-name", "", kube.LocalHelmRepoName, "The name of the helm repository for the installed Chart Museum")
	cmd.Flags().BoolVarP(&flags.NoDefaultEnvironments, "no-default-environments", "", false, "Disables the creation of the default Staging and Production environments")
	cmd.Flags().StringVarP(&flags.DefaultEnvironmentPrefix, "default-environment-prefix", "", "", "Default environment repo prefix, your Git repos will be of the form 'environment-$prefix-$envName'")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "", "jx", "The namespace the Jenkins X platform should be installed into")
	cmd.Flags().StringVarP(&flags.Timeout, "timeout", "", defaultInstallTimeout, "The number of seconds to wait for the helm install to complete")
	cmd.Flags().StringVarP(&flags.EnvironmentGitOwner, "environment-git-owner", "", "", "The Git provider organisation to create the environment Git repositories in")
	cmd.Flags().BoolVarP(&flags.RegisterLocalHelmRepo, "register-local-helmrepo", "", false, "Registers the Jenkins X ChartMuseum registry with your helm client [default false]")
	cmd.Flags().BoolVarP(&flags.CleanupTempFiles, "cleanup-temp-files", "", true, "Cleans up any temporary values.yaml used by helm install [default true]")
	cmd.Flags().BoolVarP(&flags.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	cmd.Flags().BoolVarP(&flags.InstallOnly, "install-only", "", false, "Force the install command to fail if there is already an installation. Otherwise lets update the installation")
	cmd.Flags().StringVarP(&flags.DockerRegistry, "docker-registry", "", "", "The Docker Registry host or host:port which is used when tagging and pushing images. If not specified it defaults to the internal registry unless there is a better provider default (e.g. ECR on AWS/EKS)")
	cmd.Flags().StringVarP(&flags.ExposeControllerPathMode, "exposecontroller-pathmode", "", "", "The ExposeController path mode for how services should be exposed as URLs. Defaults to using subnets. Use a value of `path` to use relative paths within the domain host such as when using AWS ELB host names")
	cmd.Flags().StringVarP(&flags.Version, "version", "", "", "The specific platform version to install")
	cmd.Flags().BoolVarP(&flags.Prow, "prow", "", false, "Enable Prow")

	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)
	options.HelmValuesConfig.AddExposeControllerValues(cmd, true)
	options.AdminSecretsService.AddAdminSecretsValues(cmd)
	options.InitOptions.addInitFlags(cmd)
}

func (flags *InstallFlags) addCloudEnvOptions(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flags.CloudEnvRepository, "cloud-environment-repo", "", DEFAULT_CLOUD_ENVIRONMENTS_URL, "Cloud Environments Git repo")
	cmd.Flags().BoolVarP(&flags.LocalCloudEnvironment, "local-cloud-environment", "", false, "Ignores default cloud-environment-repo and uses current directory ")
}

// Run implements this command
func (options *InstallOptions) Run() error {
	originalGitUsername := options.GitRepositoryOptions.Username
	originalGitServer := options.GitRepositoryOptions.ServerURL
	originalGitToken := options.GitRepositoryOptions.ApiToken

	if options.Flags.Provider == EKS {
		var deps []string
		d := binaryShouldBeInstalled("eksctl")
		if d != "" {
			deps = append(deps, d)
		}
		d = binaryShouldBeInstalled("heptio-authenticator-aws")
		if d != "" {
			deps = append(deps, d)
		}
		err := options.installMissingDependencies(deps)
		if err != nil {
			log.Errorf("%v\nPlease fix the error or install manually then try again", err)
			os.Exit(-1)
		}
	}

	client, originalNs, err := options.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the kube client")
	}
	options.KubeClientCached = client

	initOpts := &options.InitOptions
	helmBinary := initOpts.HelmBinary()

	// configure the Helm binary
	options.Helm().SetHelmBinary(helmBinary)
	if initOpts.Flags.NoTiller {
		helmer := options.Helm()
		helmCli, ok := helmer.(*helm.HelmCLI)
		if ok && helmCli != nil {
			options.helm = helm.NewHelmTemplate(helmCli, helmCli.CWD, client)
		} else {
			helmTemplate, ok := helmer.(*helm.HelmTemplate)
			if ok {
				options.helm = helmTemplate
			} else {
				log.Warnf("Helm facade is not a *helm.HelmCLI or *helm.HelmTemplate: %#v\n", helmer)
			}
		}
	}

	dependencies := []string{}
	if !initOpts.Flags.RemoteTiller {
		binDir, err := util.JXBinLocation()
		if err != nil {
			return errors.Wrap(err, "reading jx bin location")
		}
		_, install, err := shouldInstallBinary("tiller")
		if !install && err == nil {
			confirm := &survey.Confirm{
				Message: "Uninstalling  existing tiller binary:",
				Default: true,
			}
			flag := true
			err = survey.AskOne(confirm, &flag, nil)
			if err != nil || flag == false {
				return errors.New("Existing tiller must be uninstalled first in order to use the jx in tiller less mode")
			}
			// Uninstall helm and tiller first to avoid using some older version
			err = options.UninstallBinary(binDir, "tiller")
			if err != nil {
				return errors.Wrap(err, "uninstalling existing tiller binary")
			}
		}

		_, install, err = shouldInstallBinary(helmBinary)
		if !install && err == nil {
			confirm := &survey.Confirm{
				Message: "Uninstalling  existing helm binary:",
				Default: true,
			}
			flag := true
			err = survey.AskOne(confirm, &flag, nil)
			if err != nil || flag == false {
				return errors.New("Existing helm must be uninstalled first in order to use the jx in tiller less mode")
			}
			// Uninstall helm and tiller first to avoid using some older version
			err = options.UninstallBinary(binDir, helmBinary)
			if err != nil {
				return errors.Wrap(err, "uninstalling existing helm binary")
			}
		}

		dependencies = append(dependencies, "tiller")
		options.Helm().SetHost(options.tillerAddress())
	}
	dependencies = append(dependencies, helmBinary)
	err = options.installRequirements(options.Flags.Provider, dependencies...)
	if err != nil {
		return errors.Wrap(err, "failed to install the platform requirements")
	}

	context := ""
	if !options.Flags.DisableSetKubeContext {
		context, err = options.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return errors.Wrap(err, "failed to retrieve the current context from kube configuration")
		}
	}

	ns := options.Flags.Namespace
	if ns == "" {
		ns = originalNs
	}
	options.devNamespace = ns

	namespaceLabels := map[string]string{kube.LabelTeam: ns, kube.LabelEnvironment: kube.LabelValueDevEnvironment}
	err = kube.EnsureNamespaceCreated(client, ns, namespaceLabels, nil)
	if err != nil {
		return fmt.Errorf("Failed to ensure the namespace %s is created: %s\nIs this an RBAC issue on your cluster?", ns, err)
	}

	if !options.Flags.DisableSetKubeContext {
		err = options.RunCommand("kubectl", "config", "set-context", context, "--namespace", ns)
		if err != nil {
			return errors.Wrapf(err, "failed to set the context '%s' in kube configuration", context)
		}
	}

	options.Flags.Provider, err = options.GetCloudProvider(options.Flags.Provider)
	if err != nil {
		return errors.Wrapf(err, "failed to get the cloud provider '%s'", options.Flags.Provider)
	}

	initOpts.Flags.Provider = options.Flags.Provider
	initOpts.Flags.Namespace = options.Flags.Namespace
	exposeController := options.CreateEnvOptions.HelmValuesConfig.ExposeController
	initOpts.Flags.Http = true
	if exposeController != nil {
		initOpts.Flags.Http = exposeController.Config.HTTP == "true"
	}
	initOpts.BatchMode = options.BatchMode

	if options.Flags.Provider == AKS {
		/**
		 * create a cluster admin role
		 */
		err = options.createClusterAdmin()
		if err != nil {
			return errors.Wrap(err, "failed to create the cluster admin")
		}
		log.Success("created role cluster-admin")
	}

	// lets ignore errors getting the current context in case we are running inside a pod
	currentContext := ""
	if !options.Flags.DisableSetKubeContext {
		currentContext, err = options.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return errors.Wrap(err, "failed to get the current context")
		}
	}
	isAwsProvider := options.Flags.Provider == AWS || options.Flags.Provider == EKS
	if isAwsProvider {
		err = options.ensureDefaultStorageClass(client, "gp2", "kubernetes.io/aws-ebs", "gp2")
		if err != nil {
			return err
		}
	}

	if currentContext == "minikube" {
		if options.Flags.Provider == "" {
			options.Flags.Provider = MINIKUBE
		}
		ip, err := options.getCommandOutput("", "minikube", "ip")
		if err != nil {
			return errors.Wrap(err, "failed to get the IP from Minikube")
		}
		options.Flags.Domain = ip + ".nip.io"
	}

	if initOpts.Flags.Domain == "" && options.Flags.Domain != "" {
		initOpts.Flags.Domain = options.Flags.Domain
	}

	// lets default the helm domain
	if exposeController != nil {
		ecConfig := &exposeController.Config
		if ecConfig.Domain == "" && options.Flags.Domain != "" {
			ecConfig.Domain = options.Flags.Domain
			log.Success("set exposeController Config Domain " + ecConfig.Domain + "\n")
		}
		if ecConfig.PathMode == "" && options.Flags.ExposeControllerPathMode != "" {
			ecConfig.PathMode = options.Flags.ExposeControllerPathMode
			log.Success("set exposeController Config PathMode " + ecConfig.PathMode + "\n")
		}
		if ecConfig.Domain == "" && options.Flags.Domain != "" {
			ecConfig.Domain = options.Flags.Domain
			log.Success("set exposeController Config Domain " + ecConfig.Domain + "\n")
		}
		if isOpenShiftProvider(options.Flags.Provider) {
			ecConfig.Exposer = "Route"
		}
	}

	callback := func(env *v1.Environment) error {
		if env.Spec.TeamSettings.KubeProvider == "" {
			env.Spec.TeamSettings.KubeProvider = options.Flags.Provider
			log.Infof("Storing the kubernetes provider %s in the TeamSettings\n", env.Spec.TeamSettings.KubeProvider)
		}
		return nil
	}
	err = options.ModifyDevEnvironment(callback)
	if err != nil {
		return err
	}
	if initOpts.Flags.NoTiller {
		callback := func(env *v1.Environment) error {
			env.Spec.TeamSettings.HelmTemplate = true
			log.Info("Enabling helm template mode in the TeamSettings\n")
			return nil
		}
		err = options.ModifyDevEnvironment(callback)
		if err != nil {
			return err
		}
		initOpts.helm = nil
	}

	if !initOpts.Flags.RemoteTiller {
		err = options.restartLocalTiller()
		if err != nil {
			return err
		}
		initOpts.helm = options.helm
	}

	err = initOpts.Run()
	if err != nil {
		return errors.Wrap(err, "failed to initialize the jx")
	}

	if isOpenShiftProvider(options.Flags.Provider) {
		err = options.enableOpenShiftSCC(ns)
		if err != nil {
			return errors.Wrap(err, "failed to enable the OpenShiftSCC")
		}
	}

	if options.Flags.Provider == IKS {
		/**
		* Add the IBM chart repo for the Block Storage Driver
		 */
		err = options.addHelmBinaryRepoIfMissing(DEFAULT_IBMREPO_URL, "ibm")
		if err != nil {
			return errors.Wrap(err, "failed to add the IBM helm repo")
		}
		err = options.Helm().UpdateRepo()
		if err != nil {
			return errors.Wrap(err, "failed to update the helm repo")
		}
		err = options.Helm().UpgradeChart("ibm/ibmcloud-block-storage-plugin", "ibmcloud-block-storage-plugin", "default", nil, true, nil, false, false, nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to install/upgrade the IBM Cloud Block Storage drivers")
		}
		err = options.changeDefaultStorageClass(client, "ibmc-block-bronze")
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
		return errors.Wrap(err, "failed to read the git secrets from configuration")
	}

	err = options.AdminSecretsService.NewAdminSecretsConfig()
	if err != nil {
		return errors.Wrap(err, "failed to create the admin secret config service")
	}

	adminSecrets, err := options.AdminSecretsService.Secrets.String()
	if err != nil {
		return errors.Wrap(err, "failed to read the admin secrets")
	}

	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	if helmConfig.ExposeController.Config.Domain == "" {
		helmConfig.ExposeController.Config.Domain = options.InitOptions.Flags.Domain
	}
	domain := helmConfig.ExposeController.Config.Domain
	if domain != "" && addon.IsAddonEnabled("gitea") {
		helmConfig.Jenkins.Servers.GetOrCreateFirstGitea().Url = "http://gitea-gitea." + ns + "." + domain
	}
	dockerRegistry, err := options.dockerRegistryValue()
	if err != nil {
		return errors.Wrap(err, "failed to get the docker registry value")
	}
	if options.Flags.Provider == IKS {
		dockerRegistry = iks.GetClusterRegistry(client)
		helmConfig.PipelineSecrets.DockerConfig, err = iks.GetRegistryConfigJSON(dockerRegistry)
	}
	if dockerRegistry != "" {
		if helmConfig.Jenkins.Servers.Global.EnvVars == nil {
			helmConfig.Jenkins.Servers.Global.EnvVars = map[string]string{}
		}
		helmConfig.Jenkins.Servers.Global.EnvVars["DOCKER_REGISTRY"] = dockerRegistry
		if isOpenShiftProvider(options.Flags.Provider) && dockerRegistry == "docker-registry.default.svc:5000" {
			options.enableOpenShiftRegistryPermissions(ns, helmConfig, dockerRegistry)
		}
	}

	if initOpts.Flags.TillerNamespace != "" {
		if helmConfig.Jenkins.Servers.Global.EnvVars == nil {
			helmConfig.Jenkins.Servers.Global.EnvVars = map[string]string{}
		}
		helmConfig.Jenkins.Servers.Global.EnvVars["TILLER_NAMESPACE"] = initOpts.Flags.TillerNamespace
		os.Setenv("TILLER_NAMESPACE", initOpts.Flags.TillerNamespace)
	}

	// lets add any GitHub Enterprise servers
	gitAuthCfg, err := options.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create the git auth config service")
	}
	err = options.addGitServersToJenkinsConfig(helmConfig, gitAuthCfg)
	if err != nil {
		return errors.Wrap(err, "failed to add the Git servers to Jenkins config")
	}

	username := originalGitUsername
	if username == "" {
		if os.Getenv(JX_GIT_USER) != "" {
			username = os.Getenv(JX_GIT_USER)
		}
	}
	if username != "" && originalGitToken != "" && originalGitServer != "" {
		err = gitAuthCfg.SaveUserAuth(originalGitServer, &auth.UserAuth{
			ApiToken: originalGitToken,
			Username: username,
		})
		if err != nil {
			return err
		}
		log.Infof("Saving Git token configuration for server %s and user name %s.\n",
			util.ColorInfo(originalGitServer), util.ColorInfo(username))
	}

	config, err := helmConfig.String()
	if err != nil {
		return errors.Wrap(err, "failed to get the helm config")
	}

	// clone the environments repo
	wrkDir, err := options.cloneJXCloudEnvironmentsRepo()
	if err != nil {
		return errors.Wrap(err, "failed to clone the jx cloud environments repo")
	}

	// run  helm install setting the token and domain values
	if options.Flags.Provider == "" {
		return fmt.Errorf("no Kubernetes provider found to match cloud-environment with")
	}

	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(options.Flags.Provider)))
	if _, err := os.Stat(wrkDir); os.IsNotExist(err) {
		return fmt.Errorf("cloud environment dir %s not found", makefileDir)
	}

	// create a temporary file that's used to pass current git creds to helm in order to create a secret for pipelines to tag releases
	dir, err := util.ConfigDir()
	if err != nil {
		return errors.Wrap(err, "failed to create a temporary config dir for Git credentials")
	}

	secretsFileName := filepath.Join(dir, GitSecretsFile)
	err = ioutil.WriteFile(secretsFileName, []byte(secrets), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the git secrets in the secrets file")
	}

	adminSecretsFileName := filepath.Join(dir, AdminSecretsFile)
	err = ioutil.WriteFile(adminSecretsFileName, []byte(adminSecrets), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the admin secrets in the secrets file")
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the config file")
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
	secretResources := options.KubeClientCached.CoreV1().Secrets(ns)
	oldSecret, err := secretResources.Get(JXInstallConfig, metav1.GetOptions{})
	if oldSecret == nil || err != nil {
		_, err = secretResources.Create(jxSecrets)
		if err != nil {
			return errors.Wrap(err, "failed to create the jx secret resource")
		}
	} else {
		oldSecret.Data = jxSecrets.Data
		_, err = secretResources.Update(oldSecret)
		if err != nil {
			return errors.Wrap(err, "failed to update the jx secret resource")
		}
	}

	log.Infof("Generated helm values %s\n", util.ColorInfo(configFileName))

	timeout := options.Flags.Timeout
	if timeout == "" {
		timeout = defaultInstallTimeout
	}

	if !options.BatchMode && !options.Flags.Prow {
		jenkinsInstallOptions := []string{
			ServerlessJenkins,
			StaticMasterJenkins,
		}
		jenkinsInstallOption, err := util.PickNameWithDefault(jenkinsInstallOptions, "Select Jenkins installation type:", StaticMasterJenkins,
			options.In, options.Out, options.Err)
		if err != nil {
			return errors.Wrap(err, "picking Jenkins installation type")
		}
		if jenkinsInstallOption == ServerlessJenkins {
			options.Flags.Prow = true
		}
	}

	log.Infof("Installing Jenkins X platform helm chart from: %s\n", makefileDir)

	options.Verbose = true
	err = options.addHelmBinaryRepoIfMissing(DEFAULT_CHARTMUSEUM_URL, "jenkins-x")
	if err != nil {
		return errors.Wrap(err, "failed to add the jenkinx-x helm repo")
	}

	version := options.Flags.Version
	if version == "" {
		version, err = LoadVersionFromCloudEnvironmentsDir(wrkDir)
		if err != nil {
			return errors.Wrap(err, "failed to load version from cloud environments dir")
		}
	}

	err = options.Helm().UpdateRepo()
	if err != nil {
		return errors.Wrap(err, "failed to update the helm repo")
	}

	cloudEnvironmentValuesLocation := filepath.Join(makefileDir, CloudEnvValuesFile)
	cloudEnvironmentSecretsLocation := filepath.Join(makefileDir, CloudEnvSecretsFile)
	valueFiles := []string{cloudEnvironmentValuesLocation, cloudEnvironmentSecretsLocation, secretsFileName, adminSecretsFileName, configFileName}
	valueFiles, err = helm.AppendMyValues(valueFiles)
	if err != nil {
		return errors.Wrap(err, "failed to append the myvalues.yaml file")
	}

	options.currentNamespace = ns
	if options.Flags.Prow {
		// install Prow into the new env
		err = options.installProw()
		if err != nil {
			return fmt.Errorf("failed to install Prow: %v", err)
		}
	}

	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		return errors.Wrap(err, "failed to convert the helm install timeout value")
	}
	options.Helm().SetCWD(makefileDir)
	jxChart := "jenkins-x/jenkins-x-platform"
	jxRelName := "jenkins-x"

	log.Infof("Installing jx into namespace %s\n", util.ColorInfo(ns))
	if !initOpts.Flags.NoTiller {
		// Need to check the tiller pod is ready before proceeding
		log.Infof("Waiting for %s pod to be ready\n", util.ColorInfo("tiller"))
		serviceAccountName := "tiller"
		tillerNamespace := options.InitOptions.Flags.TillerNamespace

		clusterRoleBindingName := serviceAccountName
		role := options.InitOptions.Flags.TillerClusterRole

		err = options.ensureClusterRoleBinding(clusterRoleBindingName, role, tillerNamespace, serviceAccountName)
		if err != nil {
			return errors.Wrap(err, "tiller cluster role not defined")
		}
		err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
		if err != nil {
			return errors.Wrap(err, "tiller pod is not running after 10 minutes")
		}
	}

	if !options.Flags.InstallOnly {
		err = options.Helm().UpgradeChart(jxChart, jxRelName, ns, &version, true, &timeoutInt, false, false, nil, valueFiles)
	} else {
		err = options.Helm().InstallChart(jxChart, jxRelName, ns, &version, &timeoutInt, nil, valueFiles)
	}
	if err != nil {
		return errors.Wrap(err, "failed to install/upgrade the jenkins-x platform chart")
	}

	if options.Flags.CleanupTempFiles {
		err = os.Remove(secretsFileName)
		if err != nil {
			return errors.Wrap(err, "failed to cleanup the secrets file")
		}

		err = os.Remove(configFileName)
		if err != nil {
			return errors.Wrap(err, "failed to cleanup the config file")
		}
	}

	tls, err := strconv.ParseBool(exposeController.Config.TLSAcme)
	if err != nil {
		return fmt.Errorf("failed to parse TLS exposecontroller boolean %v", err)
	}
	ic := kube.IngressConfig{
		Domain:  domain,
		TLS:     tls,
		Exposer: exposeController.Config.Exposer,
	}
	// save ingress config details to a configmap
	_, err = kube.SaveAsConfigMap(options.KubeClientCached, kube.IngressConfigConfigmap, ns, ic)
	if err != nil {
		return err
	}

	// save cluster config CA and server url to a configmap
	if !options.Flags.DisableSetKubeContext {
		kubeConfig, _, err := kube.LoadConfig()
		if err != nil {
			return err
		}

		var jxInstallConfig *kube.JXInstallConfig
		if kubeConfig != nil {
			kubeConfigContext := kube.CurrentContext(kubeConfig)
			if kubeConfigContext != nil {
				server := kube.Server(kubeConfig, kubeConfigContext)
				certificateAuthorityData := kube.CertificateAuthorityData(kubeConfig, kubeConfigContext)
				jxInstallConfig = &kube.JXInstallConfig{
					Server: server,
					CA:     certificateAuthorityData,
				}
			}
		}

		_, err = kube.SaveAsConfigMap(options.KubeClientCached, kube.ConfigMapNameJXInstallConfig, ns, jxInstallConfig)
		if err != nil {
			return err
		}
	}

	err = options.waitForInstallToBeReady(ns)
	if err != nil {
		return errors.Wrap(err, "failed to wait for jenkinx-x chart installation to be ready")
	}
	log.Infof("Jenkins X deployments ready in namespace %s\n", ns)

	if options.Flags.Prow {
		callback := func(env *v1.Environment) error {
			env.Spec.WebHookEngine = v1.WebHookEngineProw
			settings := &env.Spec.TeamSettings
			settings.PromotionEngine = v1.PromotionEngineProw
			if settings.BuildPackURL == "" {
				settings.BuildPackURL = JenkinsBuildPackURL
			}
			settings.BuildPackRef = defaultProwBuildPackRef
			log.Info("Configuring the TeamSettings for Prow\n")
			return nil
		}
		err = options.ModifyDevEnvironment(callback)
		if err != nil {
			return err
		}
	}
	if !initOpts.Flags.RemoteTiller {
		callback := func(env *v1.Environment) error {
			env.Spec.TeamSettings.NoTiller = true
			log.Info("Disabling the server side use of tiller in the TeamSettings\n")
			return nil
		}
		err = options.ModifyDevEnvironment(callback)
		if err != nil {
			return err
		}
	}
	if helmBinary != "helm" {
		// default apps to use helm3 too
		helmOptions := EditHelmBinOptions{}
		helmOptions.CommonOptions = options.CommonOptions
		helmOptions.CommonOptions.BatchMode = true
		helmOptions.CommonOptions.Args = []string{helmBinary}
		helmOptions.currentNamespace = ns
		helmOptions.devNamespace = ns
		err = helmOptions.Run()
		if err != nil {
			return errors.Wrap(err, "failed to edit the helm options")
		}
	}

	addonConfig, err := addon.LoadAddonsConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load the addons configuration")
	}

	for _, ac := range addonConfig.Addons {
		if ac.Enabled {
			err = options.installAddon(ac.Name)
			if err != nil {
				return fmt.Errorf("failed to install addon %s: %s", ac.Name, err)
			}
		}
	}

	options.logAdminPassword()

	log.Info("Getting Jenkins API Token\n")
	err = options.retry(3, 2*time.Second, func() (err error) {
		options.CreateJenkinsUserOptions.CommonOptions = options.CommonOptions
		options.CreateJenkinsUserOptions.Password = options.AdminSecretsService.Flags.DefaultAdminPassword
		options.CreateJenkinsUserOptions.UseBrowser = true
		if options.BatchMode {
			options.CreateJenkinsUserOptions.BatchMode = true
			options.CreateJenkinsUserOptions.Headless = true
			log.Info("Attempting to find the Jenkins API Token with the browser in headless mode...")
		}
		err = options.CreateJenkinsUserOptions.Run()
		return
	})
	if err != nil {
		return errors.Wrap(err, "failed to get the Jenkins API token")
	}

	jxClient, _, err := options.JXClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the jx client")
	}

	err = options.updateJenkinsURL([]string{ns})
	if err != nil {
		log.Warnf("failed to update the Jenkins external URL")
	}

	if !options.Flags.NoDefaultEnvironments {
		// lets only recreate the environments if its the first time we run this
		_, envNames, err := kube.GetEnvironments(jxClient, ns)
		if err != nil || len(envNames) <= 1 {

			if options.Flags.DefaultEnvironmentPrefix == "" {
				options.Flags.DefaultEnvironmentPrefix = strings.ToLower(randomdata.SillyName())
			}

			log.Info("Creating default staging and production environments\n")
			// Common CreateEnv Options
			options.CreateEnvOptions.GitRepositoryOptions = options.GitRepositoryOptions
			options.CreateEnvOptions.GitRepositoryOptions.Owner = options.Flags.EnvironmentGitOwner
			options.CreateEnvOptions.Prefix = options.Flags.DefaultEnvironmentPrefix
			options.CreateEnvOptions.Prow = options.Flags.Prow
			if options.BatchMode {
				options.CreateEnvOptions.BatchMode = options.BatchMode
			}

			options.CreateEnvOptions.Options.Name = "staging"
			options.CreateEnvOptions.Options.Spec.Label = "Staging"
			options.CreateEnvOptions.Options.Spec.Order = 100
			err = options.CreateEnvOptions.Run()
			if err != nil {
				return errors.Wrapf(err, "failed to create staging environment in namespace %s", options.devNamespace)
			}
			options.CreateEnvOptions.Options.Name = "production"
			options.CreateEnvOptions.Options.Spec.Label = "Production"
			options.CreateEnvOptions.Options.Spec.Order = 200
			options.CreateEnvOptions.Options.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual
			options.CreateEnvOptions.PromotionStrategy = string(v1.PromotionStrategyTypeManual)

			err = options.CreateEnvOptions.Run()
			if err != nil {
				return errors.Wrapf(err, "failed to create the production environment in namespace %s", options.devNamespace)
			}
		}
	}

	err = options.saveChartmuseumAuthConfig()
	if err != nil {
		return errors.Wrap(err, "failed to save the auth config for Chartmuseum")
	}

	if options.Flags.RegisterLocalHelmRepo {
		err = options.registerLocalHelmRepo(options.Flags.LocalHelmRepoName, ns)
		if err != nil {
			return errors.Wrapf(err, "failed to register the local helm repo '%s'", options.Flags.LocalHelmRepoName)
		}
	}

	log.Success("\nJenkins X installation completed successfully\n")

	options.logAdminPassword()

	log.Infof("\nYour Kubernetes context is now set to the namespace: %s \n", util.ColorInfo(ns))
	log.Infof("To switch back to your original namespace use: %s\n", util.ColorInfo("jx ns "+originalNs))
	log.Infof("For help on switching contexts see: %s\n\n", util.ColorInfo("https://jenkins-x.io/developing/kube-context/"))

	log.Infof("To import existing projects into Jenkins:       %s\n", util.ColorInfo("jx import"))
	log.Infof("To create a new Spring Boot microservice:       %s\n", util.ColorInfo("jx create spring -d web -d actuator"))
	log.Infof("To create a new microservice from a quickstart: %s\n", util.ColorInfo("jx create quickstart"))
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

func (options *InstallOptions) enableOpenShiftSCC(ns string) error {
	log.Infof("Enabling anyuid for the Jenkins service account in namespace %s\n", ns)
	err := options.RunCommand("oc", "adm", "policy", "add-scc-to-user", "anyuid", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	err = options.RunCommand("oc", "adm", "policy", "add-scc-to-user", "hostaccess", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	err = options.RunCommand("oc", "adm", "policy", "add-scc-to-user", "privileged", "system:serviceaccount:"+ns+":jenkins")
	if err != nil {
		return err
	}
	// try fix monocular
	return options.RunCommand("oc", "adm", "policy", "add-scc-to-user", "anyuid", "system:serviceaccount:"+ns+":default")
}

func (options *InstallOptions) enableOpenShiftRegistryPermissions(ns string, helmConfig *config.HelmValuesConfig, dockerRegistry string) error {
	log.Infof("Enabling permissions for OpenShift registry in namespace %s\n", ns)
	// Open the registry so any authenticated user can pull images from the jx namespace
	err := options.RunCommand("oc", "adm", "policy", "add-role-to-group", "system:image-puller", "system:authenticated", "-n", ns)
	if err != nil {
		return err
	}
	err = options.ensureServiceAccount(ns, "jenkins-x-registry")
	if err != nil {
		return err
	}
	err = options.RunCommand("oc", "adm", "policy", "add-cluster-role-to-user", "registry-admin", "system:serviceaccount:"+ns+":jenkins-x-registry")
	if err != nil {
		return err
	}
	registryToken, err := options.getCommandOutput("", "oc", "serviceaccounts", "get-token", "jenkins-x-registry", "-n", ns)
	if err != nil {
		return err
	}
	helmConfig.PipelineSecrets.DockerConfig = `{"auths": {"` + dockerRegistry + `": {"auth": "` + base64.StdEncoding.EncodeToString([]byte("serviceaccount:"+registryToken)) + `"}}}`
	return nil
}

func (options *InstallOptions) logAdminPassword() {
	astrix := `
	
	********************************************************
	
	     NOTE: %s
	
	********************************************************
	
	`
	log.Infof(astrix+"\n", fmt.Sprintf("Your admin password is: %s", util.ColorInfo(options.AdminSecretsService.Flags.DefaultAdminPassword)))
}

// LoadVersionFromCloudEnvironmentsDir loads a version from the cloud environments directory
func LoadVersionFromCloudEnvironmentsDir(wrkDir string) (string, error) {
	version := ""
	path := filepath.Join(wrkDir, "Makefile")
	exists, err := util.FileExists(path)
	if err != nil {
		return version, err
	}
	if !exists {
		return version, fmt.Errorf("File does not exist %s", path)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return version, err
	}
	prefix := "CHART_VERSION"
	separator := ":="
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			text := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if !strings.HasPrefix(text, separator) {
				log.Warnf("expecting separator %s for line: %s in file %s", separator, line, path)
			} else {
				version = strings.TrimSpace(strings.TrimPrefix(text, separator))
				return version, nil
			}
		}
	}
	log.Warnf("File %s does not include a line starting with %s", path, prefix)
	return version, nil
}

// clones the jenkins-x cloud-environments repo to a local working dir
func (options *InstallOptions) cloneJXCloudEnvironmentsRepo() (string, error) {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	configDir, err := util.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("error determining config dir %v", err)
	}
	wrkDir := filepath.Join(configDir, "cloud-environments")
	log.Infof("Current configuration dir: %s\n", configDir)
	log.Infof("options.Flags.CloudEnvRepository: %s\n", options.Flags.CloudEnvRepository)
	log.Infof("options.Flags.LocalCloudEnvironment: %t\n", options.Flags.LocalCloudEnvironment)
	if options.Flags.LocalCloudEnvironment {
		currentDir, err := os.Getwd()
		if err != nil {
			return wrkDir, fmt.Errorf("error getting current working directory %v", err)
		}
		log.Infof("Copying local dir %s to %s\n", currentDir, wrkDir)

		return wrkDir, util.CopyDir(currentDir, wrkDir, true)
	}
	if options.Flags.CloudEnvRepository == "" {
		options.Flags.CloudEnvRepository = DEFAULT_CLOUD_ENVIRONMENTS_URL
	}
	log.Infof("Cloning the Jenkins X cloud environments repo to %s\n", wrkDir)
	_, err = git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           options.Flags.CloudEnvRepository,
		ReferenceName: "refs/heads/master",
		SingleBranch:  true,
		Progress:      options.Out,
	})
	if err != nil {
		if strings.Contains(err.Error(), "repository already exists") {
			flag := false
			if options.BatchMode {
				flag = true
			} else {
				confirm := &survey.Confirm{
					Message: "A local Jenkins X cloud environments repository already exists, recreate with latest?",
					Default: true,
				}
				err := survey.AskOne(confirm, &flag, nil, surveyOpts)
				if err != nil {
					return wrkDir, err
				}
			}
			if flag {
				err := os.RemoveAll(wrkDir)
				if err != nil {
					return wrkDir, err
				}

				return options.cloneJXCloudEnvironmentsRepo()
			}
		} else {
			return wrkDir, err
		}
	}
	return wrkDir, nil
}

// returns secrets that are used as values during the helm install
func (options *InstallOptions) getGitSecrets() (string, error) {

	// TODO JR convert to a struct and add the equivelent of the below to the secrets to enable Prow
	//helmConfig.Prow.User = initOpts.Flags.Username
	//helmConfig.Prow.HMACtoken, err = util.RandStringBytesMaskImprSrc(41)
	//if err != nil {
	//	return fmt.Errorf("cannot create a random hmac token for Prow")
	//}
	//userAuth, err := options.getGitUser("Git user to send webhook events as")
	//helmConfig.Prow.OAUTHtoken = userAuth.ApiToken
	//if err != nil {
	//	return fmt.Errorf("cannot get git token used for Prow")
	//}

	username, token, err := options.getGitToken()
	if err != nil {
		return "", err
	}

	server := options.GitRepositoryOptions.ServerURL
	if server == "" {
		return "", fmt.Errorf("No Git Server found")
	}
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")

	url := fmt.Sprintf("%s:%s@%s", username, token, server)

	pipelineSecrets := `
PipelineSecrets:
  GitCreds: |-
    https://%s
    http://%s`
	return fmt.Sprintf(pipelineSecrets, url, url), nil
}

// returns the Git Token that should be used by Jenkins X to setup credentials to clone repos and creates a secret for pipelines to tag a release
func (options *InstallOptions) getGitToken() (string, string, error) {
	username := options.GitRepositoryOptions.Username
	if username == "" {
		if os.Getenv(JX_GIT_USER) != "" {
			username = os.Getenv(JX_GIT_USER)
		}
	}
	if username != "" {
		// first check git-token flag
		if options.GitRepositoryOptions.ApiToken != "" {
			return username, options.GitRepositoryOptions.ApiToken, nil
		}

		// second check for an environment variable
		if os.Getenv(JX_GIT_TOKEN) != "" {
			return username, os.Getenv(JX_GIT_TOKEN), nil
		}
	}
	log.Infof("Lets set up a Git username and API token to be able to perform CI/CD\n\n")
	userAuth, err := options.getGitUser("")
	if err != nil {
		return "", "", err
	}
	return userAuth.Username, userAuth.ApiToken, nil
}

func (options *InstallOptions) waitForInstallToBeReady(ns string) error {
	client, _, err := options.KubeClient()
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
			log.Warnf("No service called %s could be found so couldn't wire up the local auth file to talk to chart museum\n", kube.ServiceChartMuseum)
			return nil
		}
		server = config.GetOrCreateServer(url)
	} else {
		server, err = options.findServer(config, &options.ServerFlags, "ChartMuseum server", "Try installing one via: jx create team", false)
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

func (options *InstallOptions) getGitUser(message string) (*auth.UserAuth, error) {
	var userAuth *auth.UserAuth
	authConfigSvc, err := options.CreateGitAuthConfigService()
	if err != nil {
		return userAuth, err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	gitProvider := options.GitRepositoryOptions.ServerURL
	if gitProvider != "" {
		kind := gits.SaasGitKind(gitProvider)
		server = config.GetOrCreateServerName(gitProvider, "", kind)
	} else {
		server, err = config.PickServer("Which Git provider?", options.BatchMode, options.In, options.Out, options.Err)
		if err != nil {
			return userAuth, err
		}
		options.GitRepositoryOptions.ServerURL = server.URL
	}
	url := server.URL
	if message == "" {
		message = fmt.Sprintf("%s username for CI/CD pipelines:", server.Label())
	}
	userAuth, err = config.PickServerUserAuth(server, message, options.BatchMode, "", options.In, options.Out, options.Err)
	if err != nil {
		return userAuth, err
	}
	if userAuth.IsInvalid() {
		f := func(username string) error {
			options.Git().PrintCreateRepositoryGenerateAccessToken(server, username, options.Out)
			return nil
		}

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(server.Label(), userAuth, defaultUserName, false, options.BatchMode, f, options.In, options.Out, options.Err)
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
		callback := func(env *v1.Environment) error {
			teamSettings := &env.Spec.TeamSettings
			teamSettings.GitServer = url
			teamSettings.PipelineUsername = userAuth.Username
			teamSettings.Organisation = options.Owner
			teamSettings.GitPrivate = options.GitRepositoryOptions.Private
			return nil
		}
		err = options.ModifyDevEnvironment(callback)
		if err != nil {
			return userAuth, fmt.Errorf("failed to save team settings %s", err)
		}
	}
	// TODO This API should be refactored/rethought as mixing OO and functional styles is error prone. If choosing an OO style, mutations should be carried out on the object data and then that data should be introspected as the source of truth in the operation. Alternatively, remove object state and pass values in a functional style.
	options.GitRepositoryOptions.Username = userAuth.Username
	options.GitRepositoryOptions.ApiToken = userAuth.ApiToken
	return userAuth, nil
}

func (options *InstallOptions) installAddon(name string) error {
	log.Infof("Installing addon %s\n", util.ColorInfo(name))

	opts := &CreateAddonOptions{
		CreateOptions: CreateOptions{
			CommonOptions: options.CommonOptions,
		},
		HelmUpdate: true,
	}
	if name == "gitea" {
		opts.ReleaseName = defaultGiteaReleaseName
		giteaOptions := &CreateAddonGiteaOptions{
			CreateAddonOptions: *opts,
			Chart:              kube.ChartGitea,
		}
		return giteaOptions.Run()
	}
	return opts.CreateAddon(name)
}

func (options *InstallOptions) addGitServersToJenkinsConfig(helmConfig *config.HelmValuesConfig, gitAuthCfg auth.AuthConfigService) error {
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

func (options *InstallOptions) ensureDefaultStorageClass(client kubernetes.Interface, name string, provisioner string, typeName string) error {
	storageClassInterface := client.StorageV1().StorageClasses()
	storageClasses, err := storageClassInterface.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var foundSc *storagev1.StorageClass
	for idx, sc := range storageClasses.Items {
		ann := sc.Annotations
		if ann != nil && ann[kube.AnnotationIsDefaultStorageClass] == "true" {
			return nil
		}
		if sc.Name == name {
			foundSc = &storageClasses.Items[idx]
		}
	}

	if foundSc != nil {
		// lets update the storageclass to be default
		if foundSc.Annotations == nil {
			foundSc.Annotations = map[string]string{}
		}
		foundSc.Annotations[kube.AnnotationIsDefaultStorageClass] = "true"

		log.Infof("Updating storageclass %s to be the default\n", util.ColorInfo(name))
		_, err = storageClassInterface.Update(foundSc)
		return err
	}

	// lets create a default storage class
	reclaimPolicy := core_v1.PersistentVolumeReclaimRetain

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				kube.AnnotationIsDefaultStorageClass: "true",
			},
		},
		Provisioner: provisioner,
		Parameters: map[string]string{
			"type": typeName,
		},
		ReclaimPolicy: &reclaimPolicy,
		MountOptions:  []string{"debug"},
	}
	log.Infof("Creating default storageclass %s with provisioner %s\n", util.ColorInfo(name), util.ColorInfo(provisioner))
	_, err = storageClassInterface.Create(sc)
	return err
}

func (options *InstallOptions) changeDefaultStorageClass(client kubernetes.Interface, defaultName string) error {
	storageClassInterface := client.StorageV1().StorageClasses()
	storageClasses, err := storageClassInterface.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var foundSc *storagev1.StorageClass
	for idx, sc := range storageClasses.Items {
		ann := sc.Annotations
		foundSc = &storageClasses.Items[idx]
		if sc.Name == defaultName {
			if ann == nil {
				foundSc.Annotations = map[string]string{}
			}
			foundSc.Annotations[kube.AnnotationIsDefaultStorageClass] = "true"
			_, err = storageClassInterface.Update(foundSc)
		} else {
			if ann != nil && ann[kube.AnnotationIsDefaultStorageClass] == "true" {
				foundSc.Annotations[kube.AnnotationIsDefaultStorageClass] = "false"
				_, err = storageClassInterface.Update(foundSc)
			}
		}
	}
	return nil
}

// returns the docker registry string for the given provider
func (options *InstallOptions) dockerRegistryValue() (string, error) {
	if options.Flags.DockerRegistry != "" {
		return options.Flags.DockerRegistry, nil
	}
	if options.Flags.Provider == AWS || options.Flags.Provider == EKS {
		return amazon.GetContainerRegistryHost()
	}
	if options.Flags.Provider == OPENSHIFT || options.Flags.Provider == MINISHIFT {
		return "docker-registry.default.svc:5000", nil
	}
	return "", nil
}
