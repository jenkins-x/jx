package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/vault"

	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"

	"github.com/jenkins-x/jx/pkg/addon"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cloud/aks"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/iks"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	configio "github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	git "gopkg.in/src-d/go-git.v4"
	core_v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ModifySecretCallback a callback for modifying a Secret for a given name
type ModifySecretCallback func(string, func(*core_v1.Secret) error) (*core_v1.Secret, error)

// ModifyConfigMapCallback a callback for modifying a ConfigMap for a given name
type ModifyConfigMapCallback func(string, func(*core_v1.ConfigMap) error) (*core_v1.ConfigMap, error)

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

	modifyConfigMapCallback ModifyConfigMapCallback
	modifySecretCallback    ModifySecretCallback
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
	GitOpsMode               bool
	Dir                      string
	NoGitOpsEnvApply         bool
	NoGitOpsEnvRepo          bool
	NoGitOpsVault            bool
	Vault                    bool
	BuildPackName            string
}

// Secrets struct for secrets
type Secrets struct {
	Login string
	Token string
}

const (
	JX_GIT_TOKEN = "JX_GIT_TOKEN"
	JX_GIT_USER  = "JX_GIT_USER"
	// Want to use your own provider file? Change this line to point to your fork
	DefaultCloudEnvironmentsURL = "https://github.com/jenkins-x/cloud-environments"

	// JenkinsXPlatformChartName default chart name for Jenkins X platform
	JenkinsXPlatformChartName = "jenkins-x-platform"

	// JenkinsXPlatformChart the default full chart name with the default repository prefix
	JenkinsXPlatformChart   = "jenkins-x/" + JenkinsXPlatformChartName
	JenkinsXPlatformRelease = "jenkins-x"

	AdminSecretsFile       = "adminSecrets.yaml"
	ExtraValuesFile        = "extraValues.yaml"
	JXInstallConfig        = "jx-install-config"
	CloudEnvValuesFile     = "myvalues.yaml"
	CloudEnvSecretsFile    = "secrets.yaml"
	CloudEnvSopsConfigFile = ".sops.yaml"
	defaultInstallTimeout  = "6000"

	ServerlessJenkins   = "Serverless Jenkins"
	StaticMasterJenkins = "Static Master Jenkins"

	GitOpsChartYAML = `name: env
version: 0.0.1
description: GitOps Environment for this Environment
maintainers:
  - name: Team
icon: https://www.cloudbees.com/sites/default/files/Jenkins_8.png
`

	devGitOpsGitIgnore = `
# lets not accidentally check in Secret YAMLs!
secrets.yaml
mysecrets.yaml
`

	devGitOpsReadMe = `
## Jenkins X Development Environment

This repository contains the source code for the Jenkins X Development Environment so that it can be managed via GitOps.
`

	devGitOpsJenkinsfile = `pipeline {
  agent {
    label "jenkins-jx-base"
  }
  environment {
    DEPLOY_NAMESPACE = "%s"
  }
  stages {
    stage('Validate Environment') {
      steps {
        container('jx-base') {
          dir('env') {
            sh 'jx step helm build'
          }
        }
      }
    }
    stage('Update Environment') {
      when {
        branch 'master'
      }
      steps {
        container('jx-base') {
          dir('env') {
            sh 'jx step env apply'
          }
        }
      }
    }
  }
}
`

	devGitOpsJenkinsfileProw = `pipeline {
  agent any
  environment {
    DEPLOY_NAMESPACE = "%s"
  }
  stages {
    stage('Validate Environment') {
      steps {
        dir('env') {
          sh 'jx step helm build'
        }
      }
    }
    stage('Update Environment') {
      when {
        branch 'master'
      }
      steps {
        dir('env') {
          sh 'jx step env apply'
        }
      }
    }
  }
}
`
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
	}

	options.addCommonFlags(cmd)
	options.addInstallFlags(cmd, false)

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the Kubernetes cluster.  Supported providers: "+KubernetesProviderOptions())

	cmd.AddCommand(NewCmdInstallDependencies(f, in, out, errOut))

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
				ExposeController: &config.ExposeController{
					Config: config.ExposeControllerConfig{
						HTTP:    "true",
						TLSAcme: "false",
						Exposer: "Ingress",
					},
				},
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
	cmd.Flags().StringVarP(&flags.LocalHelmRepoName, "local-helm-repo-name", "", kube.LocalHelmRepoName, "The name of the helm repository for the installed ChartMuseum")
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
	cmd.Flags().BoolVarP(&flags.GitOpsMode, "gitops", "", false, "Sets up the local file system for GitOps so that the current installation can be configured or upgraded at any time via GitOps")
	cmd.Flags().BoolVarP(&flags.NoGitOpsEnvApply, "no-gitops-env-apply", "", false, "When using GitOps to create the source code for the development environment and installation, don't run 'jx step env apply' to perform the install")
	cmd.Flags().BoolVarP(&flags.NoGitOpsEnvRepo, "no-gitops-env-repo", "", false, "When using GitOps to create the source code for the development environment this flag disables the creation of a git repository for the source code")
	cmd.Flags().BoolVarP(&flags.NoGitOpsVault, "no-gitops-vault", "", false, "When using GitOps to create the source code for the development environment this flag disables the creation of a vault")
	cmd.Flags().BoolVarP(&flags.Vault, "vault", "", false, "Sets up a Hashicorp Vault for storing secrets during installation (supported only for GKE)")
	cmd.Flags().StringVarP(&flags.BuildPackName, "buildpack", "", "", "The name of the build pack to use for the Team")

	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)
	options.HelmValuesConfig.AddExposeControllerValues(cmd, true)
	options.AdminSecretsService.AddAdminSecretsValues(cmd)
	options.InitOptions.addInitFlags(cmd)
}

func (flags *InstallFlags) addCloudEnvOptions(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flags.CloudEnvRepository, "cloud-environment-repo", "", DefaultCloudEnvironmentsURL, "Cloud Environments Git repo")
	cmd.Flags().BoolVarP(&flags.LocalCloudEnvironment, "local-cloud-environment", "", false, "Ignores default cloud-environment-repo and uses current directory ")
}

// Run implements this command
func (options *InstallOptions) Run() error {
	configStore := configio.NewFileStore()

	// Default to verbose mode to get more information during the install
	options.Verbose = true

	client, originalNs, err := options.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "creating the kube client")
	}

	ns := options.Flags.Namespace
	if ns == "" {
		ns = originalNs
	}
	options.SetDevNamespace(ns)

	err = options.registerAllCRDs()
	if err != nil {
		return errors.Wrap(err, "registering all CRDs")
	}

	gitOpsDir, gitOpsEnvDir, err := options.configureGitOpsMode(configStore, ns)
	if err != nil {
		return errors.Wrap(err, "configuring the GitOps mode")
	}

	options.configureHelm(client, originalNs)
	err = options.installHelmBinaries()
	if err != nil {
		return errors.Wrap(err, "installing helm binaries")
	}

	err = options.installCloudProviderDependencies()
	if err != nil {
		return errors.Wrap(err, "installing cloud provider dependencies")
	}

	err = options.configureKubectl(ns)
	if err != nil {
		return errors.Wrap(err, "configure the kubectl")
	}

	options.Flags.Provider, err = options.GetCloudProvider(options.Flags.Provider)
	if err != nil {
		return errors.Wrapf(err, "retrieving cloud provider '%s'", options.Flags.Provider)
	}

	err = options.setMinikubeFromContext()
	if err != nil {
		return errors.Wrap(err, "configuring minikube from kubectl context")
	}

	err = options.configureCloudProivderPreInit(client)
	if err != nil {
		return errors.Wrap(err, "configuring the cloud provider before initializing the platform")
	}

	err = options.init()
	if err != nil {
		return errors.Wrap(err, "initializing the Jenkins X platform")
	}

	err = options.configureCloudProivderPostInit(client, ns)
	if err != nil {
		return errors.Wrap(err, "configuring the cloud provider after initializing the platform")
	}

	ic, err := options.saveIngressConfig()
	if err != nil {
		return errors.Wrap(err, "saving the ingress configuration in a ConfigMap")
	}

	err = options.saveClusterConfig()
	if err != nil {
		return errors.Wrap(err, "saving the cluster configuration in a ConfigMap")
	}

	err = options.createSystemVault(client, ns, ic)
	if err != nil {
		return errors.Wrap(err, "creating the system vault")
	}

	err = options.configureGitAuth()
	if err != nil {
		return errors.Wrap(err, "configuring the git auth")
	}

	err = options.configureDockerRegistry(client, ns)
	if err != nil {
		return errors.Wrap(err, "configuring the docker registry")
	}

	cloudEnvDir, err := options.cloneJXCloudEnvironmentsRepo()
	if err != nil {
		return errors.Wrap(err, "cloning the jx cloud environments repo")
	}

	err = options.configureHelmValues(ns)
	if err != nil {
		return errors.Wrap(err, "configuring helm values")
	}

	if options.Flags.Provider == "" {
		return fmt.Errorf("no Kubernetes provider found to match cloud-environment with")
	}
	providerEnvDir := filepath.Join(cloudEnvDir, fmt.Sprintf("env-%s", strings.ToLower(options.Flags.Provider)))
	valuesFiles, secretsFiles, temporaryFiles, err := options.getHelmValuesFiles(configStore, providerEnvDir)
	if err != nil {
		return errors.Wrap(err, "getting the helm value files")
	}

	log.Infof("Installing Jenkins X platform helm chart from: %s\n", providerEnvDir)

	err = options.configureHelmRepo()
	if err != nil {
		return errors.Wrap(err, "configuring the Jenkins X helm repository")
	}

	err = options.selectJenkinsInstallation()
	if err != nil {
		return errors.Wrap(err, "selecting the Jenkins installation type")
	}

	err = options.configureAndInstallProw(ns)
	if err != nil {
		return errors.Wrap(err, "configuring and installing Prow")
	}

	err = options.verifyTiller(client, ns)
	if err != nil {
		return errors.Wrap(err, "verifying Tiller is running")
	}

	err = options.configureBuildPackMode()
	if err != nil {
		return errors.Wrap(err, "configuring the build pack mode")
	}

	log.Infof("Installing jx into namespace %s\n", util.ColorInfo(ns))

	version, err := options.getPlatformVersion(cloudEnvDir, configStore)
	if err != nil {
		return errors.Wrap(err, "getting the platform version")
	}

	if options.Flags.GitOpsMode {
		err := options.installPlatformGitOpsMode(gitOpsEnvDir, gitOpsDir, configStore, DEFAULT_CHARTMUSEUM_URL,
			JenkinsXPlatformChartName, ns, version, valuesFiles, secretsFiles)
		if err != nil {
			return errors.Wrap(err, "installing the Jenkins X platform in GitOps mode")
		}
	} else {
		err := options.installPlatform(providerEnvDir, JenkinsXPlatformChart, JenkinsXPlatformRelease,
			ns, version, valuesFiles, secretsFiles)
		if err != nil {
			return errors.Wrap(err, "installing the Jenkins X platform")
		}
	}

	if options.Flags.CleanupTempFiles {
		err := options.cleanupTempFiles(temporaryFiles)
		if err != nil {
			return errors.Wrap(err, "cleaning up the temporary files")
		}
	}

	err = options.configureProwInTeamSettings()
	if err != nil {
		return errors.Wrap(err, "configuring Prow in team settings")
	}

	err = options.configureTillerInDevEnvironment()
	if err != nil {
		return errors.Wrap(err, "configuring Tiller in the dev environment")
	}

	err = options.configureHelm3(ns)
	if err != nil {
		return errors.Wrap(err, "configuring helm3")
	}

	err = options.installAddons()
	if err != nil {
		return errors.Wrap(err, "installing the Jenkins X Addons")
	}

	options.logAdminPassword()

	err = options.configureJenkins(ns)
	if err != nil {
		return errors.Wrap(err, "configuring Jenkins")
	}

	err = options.createEnvironments(ns)
	if err != nil {
		return errors.Wrap(err, "creating the environments")
	}

	err = options.saveChartmuseumAuthConfig()
	if err != nil {
		return errors.Wrap(err, "saving the ChartMuseum auth configuration")
	}

	if options.Flags.RegisterLocalHelmRepo {
		err = options.registerLocalHelmRepo(options.Flags.LocalHelmRepoName, ns)
		if err != nil {
			return errors.Wrapf(err, "registering the local helm repo '%s'", options.Flags.LocalHelmRepoName)
		}
	}

	err = options.generateGitOpsDevEnvironmentConfig(gitOpsDir)
	if err != nil {
		return errors.Wrap(err, "generating the GitOps development environment config")
	}

	err = options.applyGitOpsDevEnvironmentConfig(gitOpsEnvDir, ns)
	if err != nil {
		return errors.Wrap(err, "applying the GitOps development environment config")
	}

	log.Successf("\nJenkins X installation completed successfully")

	options.logAdminPassword()

	log.Infof("\nYour Kubernetes context is now set to the namespace: %s \n", util.ColorInfo(ns))
	log.Infof("To switch back to your original namespace use: %s\n", util.ColorInfo("jx namespace "+originalNs))
	log.Infof("For help on switching contexts see: %s\n\n", util.ColorInfo("https://jenkins-x.io/developing/kube-context/"))

	log.Infof("To import existing projects into Jenkins:       %s\n", util.ColorInfo("jx import"))
	log.Infof("To create a new Spring Boot microservice:       %s\n", util.ColorInfo("jx create spring -d web -d actuator"))
	log.Infof("To create a new microservice from a quickstart: %s\n", util.ColorInfo("jx create quickstart"))
	return nil
}

func (options *InstallOptions) configureKubectl(namespace string) error {
	context := ""
	var err error
	if !options.Flags.DisableSetKubeContext {
		context, err = options.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return errors.Wrap(err, "failed to retrieve the current context from kube configuration")
		}
	}

	if !options.Flags.DisableSetKubeContext {
		err = options.RunCommand("kubectl", "config", "set-context", context, "--namespace", namespace)
		if err != nil {
			return errors.Wrapf(err, "failed to set the context '%s' in kube configuration", context)
		}
	}
	return nil
}

func (options *InstallOptions) init() error {
	initOpts := &options.InitOptions
	initOpts.Flags.Provider = options.Flags.Provider
	initOpts.Flags.Namespace = options.Flags.Namespace
	exposeController := options.CreateEnvOptions.HelmValuesConfig.ExposeController
	initOpts.Flags.Http = true
	if exposeController != nil {
		initOpts.Flags.Http = exposeController.Config.HTTP == "true"
	}
	initOpts.BatchMode = options.BatchMode
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
	err := options.ModifyDevEnvironment(callback)
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

	if !initOpts.Flags.RemoteTiller && !initOpts.Flags.NoTiller {
		err = restartLocalTiller()
		if err != nil {
			return err
		}
		initOpts.helm = options.helm
	}

	err = initOpts.Run()
	if err != nil {
		return errors.Wrap(err, "failed to initialize the jx")
	}

	if initOpts.Flags.Domain != "" && options.Flags.Domain == "" {
		options.Flags.Domain = initOpts.Flags.Domain
	}

	return nil
}

func (options *InstallOptions) getPlatformVersion(cloudEnvDir string,
	configStore configio.ConfigStore) (string, error) {
	version := options.Flags.Version
	var err error
	if version == "" {
		version, err = LoadVersionFromCloudEnvironmentsDir(cloudEnvDir, configStore)
		if err != nil {
			return "", errors.Wrap(err, "failed to load version from cloud environments dir")
		}
	}
	return version, nil
}

func (options *InstallOptions) installPlatform(providerEnvDir string, jxChart string, jxRelName string,
	namespace string, version string, valuesFiles []string, secretsFiles []string) error {

	options.Helm().SetCWD(providerEnvDir)

	timeout := options.Flags.Timeout
	if timeout == "" {
		timeout = defaultInstallTimeout
	}
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		return errors.Wrap(err, "failed to convert the helm install timeout value")
	}

	allValuesFiles := []string{}
	allValuesFiles = append(allValuesFiles, valuesFiles...)
	allValuesFiles = append(allValuesFiles, secretsFiles...)
	for _, f := range allValuesFiles {
		options.Debugf("Adding values file %s\n", util.ColorInfo(f))
	}

	if !options.Flags.InstallOnly {
		err = options.Helm().UpgradeChart(jxChart, jxRelName, namespace, &version, true,
			&timeoutInt, false, false, nil, allValuesFiles, "", "", "")
	} else {
		err = options.Helm().InstallChart(jxChart, jxRelName, namespace, &version, &timeoutInt,
			nil, allValuesFiles, "", "", "")
	}
	if err != nil {
		return errors.Wrap(err, "failed to install/upgrade the jenkins-x platform chart")
	}

	err = options.waitForInstallToBeReady(namespace)
	if err != nil {
		return errors.Wrap(err, "failed to wait for jenkins-x chart installation to be ready")
	}
	log.Infof("Jenkins X deployments ready in namespace %s\n", namespace)
	return nil
}

func (options *InstallOptions) installPlatformGitOpsMode(gitOpsEnvDir string, gitOpsDir string, configStore configio.ConfigStore,
	chartRepository string, chartName string, namespace string, version string, valuesFiles []string, secretsFiles []string) error {
	options.CreateEnvOptions.NoDevNamespaceInit = true
	deps := []*helm.Dependency{
		{
			Name:       JenkinsXPlatformChartName,
			Version:    version,
			Repository: DEFAULT_CHARTMUSEUM_URL,
		},
	}
	requirements := &helm.Requirements{
		Dependencies: deps,
	}

	chartFile := filepath.Join(gitOpsEnvDir, helm.ChartFileName)
	requirementsFile := filepath.Join(gitOpsEnvDir, helm.RequirementsFileName)
	secretsFile := filepath.Join(gitOpsEnvDir, helm.SecretsFileName)
	valuesFile := filepath.Join(gitOpsEnvDir, helm.ValuesFileName)
	err := helm.SaveRequirementsFile(requirementsFile, requirements)
	if err != nil {
		return errors.Wrapf(err, "failed to save GitOps helm requirements file %s", requirementsFile)
	}

	err = configStore.Write(chartFile, []byte(GitOpsChartYAML))
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", chartFile)
	}

	err = helm.CombineValueFilesToFile(secretsFile, secretsFiles, JenkinsXPlatformChartName, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to generate %s by combining helm Secret YAML files %s", secretsFile, strings.Join(secretsFiles, ", "))
	}

	if options.Flags.Vault {
		err := options.storeSecretYamlFilesInVault(vault.GitOpsSecretsPath, secretsFile)
		if err != nil {
			return errors.Wrapf(err, "storing in Vault the secrets files: %s", secretsFile)
		}

		err = util.DestroyFile(secretsFile)
		if err != nil {
			return errors.Wrapf(err, "destroying the secrets file '%s' after storing it in Vault", secretsFile)
		}
	}

	extraValues := map[string]interface{}{
		"postinstalljob": map[string]interface{}{"enabled": "true"},
	}
	err = helm.CombineValueFilesToFile(valuesFile, valuesFiles, JenkinsXPlatformChartName, extraValues)
	if err != nil {
		return errors.Wrapf(err, "failed to generate %s by combining helm value YAML files %s", valuesFile, strings.Join(valuesFiles, ", "))
	}

	gitIgnore := filepath.Join(gitOpsDir, ".gitignore")
	err = configStore.Write(gitIgnore, []byte(devGitOpsGitIgnore))
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", gitIgnore)
	}

	readme := filepath.Join(gitOpsDir, "README.md")
	err = configStore.Write(readme, []byte(devGitOpsReadMe))
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", readme)
	}

	jenkinsFile := filepath.Join(gitOpsDir, "Jenkinsfile")
	jftTmp := devGitOpsJenkinsfile
	isProw := options.Flags.Prow
	if isProw {
		jftTmp = devGitOpsJenkinsfileProw
	}
	text := fmt.Sprintf(jftTmp, namespace)
	err = configStore.Write(jenkinsFile, []byte(text))
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", jenkinsFile)
	}
	return nil
}

func (options *InstallOptions) configureAndInstallProw(namespace string) error {
	options.currentNamespace = namespace
	if options.Flags.Prow {
		_, pipelineUser, err := options.getPipelineGitAuth()
		if err != nil {
			return errors.Wrap(err, "retrieving the pipeline Git Auth")
		}
		options.OAUTHToken = pipelineUser.ApiToken
		err = options.installProw()
		if err != nil {
			errors.Wrap(err, "installing Prow")
		}
	}
	return nil
}

func (options *InstallOptions) configureHelm3(namespace string) error {
	initOpts := &options.InitOptions
	helmBinary := initOpts.HelmBinary()
	if helmBinary != "helm" {
		helmOptions := EditHelmBinOptions{}
		helmOptions.CommonOptions = options.CommonOptions
		helmOptions.CommonOptions.BatchMode = true
		helmOptions.CommonOptions.Args = []string{helmBinary}
		helmOptions.currentNamespace = namespace
		helmOptions.devNamespace = namespace
		err := helmOptions.Run()
		if err != nil {
			return errors.Wrap(err, "failed to edit the helm options")
		}
	}
	return nil
}

func (options *InstallOptions) configureHelm(client kubernetes.Interface, namespace string) {
	initOpts := &options.InitOptions
	helmBinary := initOpts.HelmBinary()
	options.Helm().SetHelmBinary(helmBinary)
	if initOpts.Flags.NoTiller {
		helmer := options.Helm()
		helmCli, ok := helmer.(*helm.HelmCLI)
		if ok && helmCli != nil {
			options.helm = helm.NewHelmTemplate(helmCli, helmCli.CWD, client, namespace)
		} else {
			helmTemplate, ok := helmer.(*helm.HelmTemplate)
			if ok {
				options.helm = helmTemplate
			} else {
				log.Warnf("Helm facade is not a *helm.HelmCLI or *helm.HelmTemplate: %#v\n", helmer)
			}
		}
	}
}

func (options *InstallOptions) configureHelmRepo() error {
	err := options.addHelmBinaryRepoIfMissing(DEFAULT_CHARTMUSEUM_URL, "jenkins-x", "", "")
	if err != nil {
		return errors.Wrap(err, "failed to add the jenkinx-x helm repo")
	}

	err = options.Helm().UpdateRepo()
	if err != nil {
		return errors.Wrap(err, "failed to update the helm repo")
	}
	return nil
}

func (options *InstallOptions) selectJenkinsInstallation() error {
	if !options.BatchMode && !options.Flags.Prow {
		jenkinsInstallOptions := []string{
			ServerlessJenkins,
			StaticMasterJenkins,
		}
		jenkinsInstallOption, err := util.PickNameWithDefault(jenkinsInstallOptions, "Select Jenkins installation type:", StaticMasterJenkins, "", options.In, options.Out, options.Err)
		if err != nil {
			return errors.Wrap(err, "picking Jenkins installation type")
		}
		if jenkinsInstallOption == ServerlessJenkins {
			options.Flags.Prow = true
		}
	}
	return nil
}

func (options *InstallOptions) configureTillerNamespace() error {
	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	initOpts := &options.InitOptions
	if initOpts.Flags.TillerNamespace != "" {
		if helmConfig.Jenkins.Servers.Global.EnvVars == nil {
			helmConfig.Jenkins.Servers.Global.EnvVars = map[string]string{}
		}
		helmConfig.Jenkins.Servers.Global.EnvVars["TILLER_NAMESPACE"] = initOpts.Flags.TillerNamespace
		os.Setenv("TILLER_NAMESPACE", initOpts.Flags.TillerNamespace)
	}
	return nil
}

func (options *InstallOptions) configureHelmValues(namespace string) error {
	helmConfig := &options.CreateEnvOptions.HelmValuesConfig

	domain := helmConfig.ExposeController.Config.Domain
	if domain != "" && addon.IsAddonEnabled("gitea") {
		helmConfig.Jenkins.Servers.GetOrCreateFirstGitea().Url = "http://gitea-gitea." + namespace + "." + domain
	}

	err := options.addGitServersToJenkinsConfig(helmConfig)
	if err != nil {
		return errors.Wrap(err, "configuring the Git Servers into Jenkins configuration")
	}

	err = options.configureTillerNamespace()
	if err != nil {
		return errors.Wrap(err, "configuring the tiller namespace")
	}

	if !options.Flags.GitOpsMode {
		options.SetDevNamespace(namespace)
	}

	isProw := options.Flags.Prow
	if isProw {
		enableJenkins := false
		helmConfig.Jenkins.Enabled = &enableJenkins
		enableControllerBuild := true
		helmConfig.ControllerBuild.Enabled = &enableControllerBuild
	}
	return nil
}

func (options *InstallOptions) getHelmValuesFiles(configStore configio.ConfigStore, providerEnvDir string) ([]string, []string, []string, error) {
	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	cloudEnvironmentValuesLocation := filepath.Join(providerEnvDir, CloudEnvValuesFile)
	cloudEnvironmentSecretsLocation := filepath.Join(providerEnvDir, CloudEnvSecretsFile)

	valuesFiles := []string{}
	secretsFiles := []string{}
	temporaryFiles := []string{}

	adminSecretsFileName, adminSecrets, err := options.getAdminSecrets(configStore,
		providerEnvDir, cloudEnvironmentSecretsLocation)
	if err != nil {
		return valuesFiles, secretsFiles, temporaryFiles,
			errors.Wrap(err, "creating the admin secrets")
	}

	dir, err := util.ConfigDir()
	if err != nil {
		return valuesFiles, secretsFiles, temporaryFiles,
			errors.Wrap(err, "creating a temporary config dir for Git credentials")
	}

	extraValuesFileName := filepath.Join(dir, ExtraValuesFile)
	err = configStore.WriteObject(extraValuesFileName, helmConfig)
	if err != nil {
		return valuesFiles, secretsFiles, temporaryFiles,
			errors.Wrapf(err, "writing the helm config in the file '%s'", extraValuesFileName)
	}
	log.Infof("Generated helm values %s\n", util.ColorInfo(extraValuesFileName))

	err = options.modifySecrets(helmConfig, adminSecrets)
	if err != nil {
		return valuesFiles, temporaryFiles, secretsFiles, errors.Wrap(err, "updating the secrets data in Kubernetes cluster")
	}

	valuesFiles = append(valuesFiles, cloudEnvironmentValuesLocation)
	valuesFiles, err = helm.AppendMyValues(valuesFiles)
	if err != nil {
		return valuesFiles, secretsFiles, temporaryFiles,
			errors.Wrap(err, "failed to append the myvalues.yaml file")
	}
	secretsFiles = append(secretsFiles,
		[]string{adminSecretsFileName, extraValuesFileName, cloudEnvironmentSecretsLocation}...)

	if options.Flags.Vault {
		temporaryFiles = append(temporaryFiles, adminSecretsFileName, extraValuesFileName, cloudEnvironmentSecretsLocation)
	} else {
		temporaryFiles = append(temporaryFiles, extraValuesFileName, cloudEnvironmentSecretsLocation)
	}

	return util.FilterFileExists(valuesFiles), util.FilterFileExists(secretsFiles), util.FilterFileExists(temporaryFiles), nil
}

func (options *InstallOptions) configureGitAuth() error {
	log.Infof("Lets set up a Git user name and API token to be able to perform CI/CD\n\n")
	gitUsername := options.GitRepositoryOptions.Username
	gitServer := options.GitRepositoryOptions.ServerURL
	gitAPIToken := options.GitRepositoryOptions.ApiToken

	if gitUsername == "" {
		gitUsernameEnv := os.Getenv(JX_GIT_USER)
		if gitUsernameEnv != "" {
			gitUsername = gitUsernameEnv
		}
	}

	if gitAPIToken == "" {
		gitAPITokenEnv := os.Getenv(JX_GIT_TOKEN)
		if gitAPITokenEnv != "" {
			gitAPIToken = gitAPITokenEnv
		}
	}

	authConfigSvc, err := options.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create the git auth config service")
	}

	authConfig := authConfigSvc.Config()
	var userAuth *auth.UserAuth
	if gitUsername != "" && gitAPIToken != "" && gitServer != "" {
		userAuth = &auth.UserAuth{
			ApiToken: gitAPIToken,
			Username: gitUsername,
		}
		authConfig.SetUserAuth(gitServer, userAuth)
	}

	var authServer *auth.AuthServer
	if gitServer != "" {
		kind := ""
		if options.GitRepositoryOptions.ServerKind == "" {
			kind = gits.SaasGitKind(gitServer)
		} else {
			kind = options.GitRepositoryOptions.ServerKind
		}
		authServer = authConfig.GetOrCreateServerName(gitServer, "", kind)
	} else {
		authServer, err = authConfig.PickServer("Which Git provider:", options.BatchMode, options.In, options.Out, options.Err)
		if err != nil {
			return errors.Wrap(err, "getting the git provider from user")
		}
	}

	message := fmt.Sprintf("local Git user for %s server:", authServer.Label())
	userAuth, err = authConfig.PickServerUserAuth(authServer, message, options.BatchMode, "", options.In, options.Out, options.Err)
	if err != nil {
		return errors.Wrapf(err, "selecting the local user for git server %s", authServer.Label())
	}

	if userAuth.IsInvalid() {
		log.Infof("Creating a local Git user for %s server\n", authServer.Label())
		f := func(username string) error {
			options.Git().PrintCreateRepositoryGenerateAccessToken(authServer, username, options.Out)
			return nil
		}
		defaultUserName := ""
		err = authConfig.EditUserAuth(authServer.Label(), userAuth, defaultUserName, false, options.BatchMode, f,
			options.In, options.Out, options.Err)
		if err != nil {
			return errors.Wrapf(err, "creating a user authentication for git server %s", authServer.Label())
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("invalid user authentication for git server %s", authServer.Label())
		}
		authConfig.SetUserAuth(gitServer, userAuth)
	}

	log.Infof("Select the CI/CD pipelines Git server and user\n")
	var pipelineAuthServer *auth.AuthServer
	if options.BatchMode {
		pipelineAuthServer = authServer
	} else {
		surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Do you wish to use %s as the pipelines Git server:", authServer.Label()),
			Default: true,
		}
		yes := false
		err = survey.AskOne(confirm, &yes, nil, surveyOpts)
		if err != nil {
			return errors.Wrap(err, "selecting pipelines Git server")
		}
		if yes {
			pipelineAuthServer = authServer
		} else {
			pipelineAuthServerURL, err := util.PickValue("Git Service URL:", gits.GitHubURL, true, "",
				options.In, options.Out, options.Err)
			if err != nil {
				return errors.Wrap(err, "reading the pipelines Git service URL")
			}
			pipelineAuthServer, err = authConfig.PickOrCreateServer(gits.GitHubURL, pipelineAuthServerURL,
				"Which Git Service do you wish to use:",
				options.BatchMode, options.In, options.Out, options.Err)
			if err != nil {
				return errors.Wrap(err, "selecting the pipelines Git Service")
			}
		}
	}

	message = fmt.Sprintf("pipelines Git user for %s server:", pipelineAuthServer.Label())
	pipelineUserAuth, err := authConfig.PickServerUserAuth(authServer, message, options.BatchMode, "", options.In, options.Out, options.Err)
	if err != nil {
		return errors.Wrapf(err, "selecting the pipeline user for git server %s", authServer.Label())
	}
	if pipelineUserAuth.IsInvalid() {
		log.Infof("Creating a pipelines Git user for %s server\n", authServer.Label())
		f := func(username string) error {
			options.Git().PrintCreateRepositoryGenerateAccessToken(pipelineAuthServer, username, options.Out)
			return nil
		}
		defaultUserName := ""
		err = authConfig.EditUserAuth(pipelineAuthServer.Label(), pipelineUserAuth, defaultUserName, false, options.BatchMode,
			f, options.In, options.Out, options.Err)
		if err != nil {
			return errors.Wrapf(err, "creating a pipeline user authentication for git server %s", authServer.Label())
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("invalid pipeline user authentication for git server %s", authServer.Label())
		}
		authConfig.SetUserAuth(pipelineAuthServer.URL, pipelineUserAuth)
	}

	pipelineAuthServerURL := pipelineAuthServer.URL
	pipelineAuthUsername := pipelineUserAuth.Username

	log.Infof("Setting the pipelines Git server %s and user name %s.\n",
		util.ColorInfo(pipelineAuthServerURL), util.ColorInfo(pipelineAuthUsername))
	authConfig.UpdatePipelineServer(pipelineAuthServer, pipelineUserAuth)

	log.Infof("Saving the Git authentication configuration")
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return errors.Wrap(err, "saving the Git authentication configuration")
	}

	editTeamSettingsCallback := func(env *v1.Environment) error {
		teamSettings := &env.Spec.TeamSettings
		teamSettings.GitServer = pipelineAuthServerURL
		teamSettings.PipelineUsername = pipelineAuthUsername
		teamSettings.Organisation = options.Owner
		teamSettings.GitPrivate = options.GitRepositoryOptions.Private
		return nil
	}
	err = options.ModifyDevEnvironment(editTeamSettingsCallback)
	if err != nil {
		return errors.Wrap(err, "updating the team settings into the environment configuration")
	}

	return nil
}

func (options *InstallOptions) buildGitRepositoryOptionsForEnvironments() (*gits.GitRepositoryOptions, error) {
	authConfigSvc, err := options.CreateGitAuthConfigService()
	if err != nil {
		return nil, errors.Wrap(err, "creating Git authentication config service")
	}
	config := authConfigSvc.Config()

	server := config.CurrentAuthServer()
	if server == nil {
		return nil, fmt.Errorf("no current git server set in the configuration")
	}
	user := config.CurrentUser(server, false)
	if user == nil {
		return nil, fmt.Errorf("no current git user set in configuration for server '%s'", server.Label())
	}

	org := options.Flags.EnvironmentGitOwner
	if org == "" {
		if options.BatchMode {
			jxClient, _, err := options.JXClientAndDevNamespace()
			if err != nil {
				return nil, errors.Wrap(err, "determining the git owner for environments")
			}
			org, err = kube.GetDevEnvGitOwner(jxClient)
			if err != nil {
				return nil, errors.Wrap(err, "determining the git owner for environments")
			}
			if org == "" {
				org = user.Username
			}

			log.Infof("Using %s environment git owner in batch mode.\n", util.ColorInfo(org))
		} else {
			provider, err := gits.CreateProvider(server, user, options.Git())
			if err != nil {
				return nil, errors.Wrap(err, "creating the Git provider")
			}

			orgs := gits.GetOrganizations(provider, user.Username)
			if len(orgs) == 0 {
				return nil, fmt.Errorf("user '%s' has no organizations", user.Username)
			}

			surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
			sort.Strings(orgs)
			prompt := &survey.Select{
				Message: "Select the organization where you want to create the environment repository:",
				Options: orgs,
			}
			err = survey.AskOne(prompt, &org, survey.Required, surveyOpts)
			if err != nil {
				return nil, errors.Wrap(err, "selecting the organization for environment repository")
			}
		}
	}

	//Save selected organisation for Environment repos.
	err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
		env.Spec.TeamSettings.EnvOrganisation = org
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "updating the TeamSettings with Environments organisation")
	}

	return &gits.GitRepositoryOptions{
		ServerURL: server.URL,
		Username:  user.Username,
		ApiToken:  user.ApiToken,
		Owner:     org,
		Private:   options.GitRepositoryOptions.Private,
	}, nil
}

func (options *InstallOptions) cleanupTempFiles(temporaryFiles []string) error {
	for _, tempFile := range temporaryFiles {
		exists, err := util.FileExists(tempFile)
		if exists && err == nil {
			err := util.DestroyFile(tempFile)
			if err != nil {
				return errors.Wrapf(err, "removing temporary file '%s'", tempFile)
			}
		}
	}
	return nil
}

func (options *InstallOptions) verifyTiller(client kubernetes.Interface, namespace string) error {
	initOpts := &options.InitOptions
	if !initOpts.Flags.NoTiller {
		serviceAccountName := "tiller"
		tillerNamespace := options.InitOptions.Flags.TillerNamespace

		log.Infof("Waiting for %s pod to be ready, service account name is %s, namespace is %s, tiller namespace is %s\n",
			util.ColorInfo("tiller"), util.ColorInfo(serviceAccountName), util.ColorInfo(namespace), util.ColorInfo(tillerNamespace))

		clusterRoleBindingName := serviceAccountName + "-role-binding"
		role := options.InitOptions.Flags.TillerClusterRole

		log.Infof("Waiting for cluster role binding to be defined, named %s in namespace %s\n ", util.ColorInfo(clusterRoleBindingName), util.ColorInfo(namespace))
		err := options.ensureClusterRoleBinding(clusterRoleBindingName, role, namespace, serviceAccountName)
		if err != nil {
			return errors.Wrap(err, "tiller cluster role not defined")
		}
		log.Infof("tiller cluster role defined: %s in namespace %s\n", util.ColorInfo(role), util.ColorInfo(namespace))

		err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
		if err != nil {
			msg := fmt.Sprintf("tiller pod (tiller-deploy in namespace %s) is not running after 10 minutes", tillerNamespace)
			return errors.Wrap(err, msg)
		}
		log.Infoln("tiller pod running")
	}
	return nil
}

func (options *InstallOptions) configureTillerInDevEnvironment() error {
	initOpts := &options.InitOptions
	if !initOpts.Flags.RemoteTiller && !initOpts.Flags.NoTiller {
		callback := func(env *v1.Environment) error {
			env.Spec.TeamSettings.NoTiller = true
			log.Info("Disabling the server side use of tiller in the TeamSettings\n")
			return nil
		}
		err := options.ModifyDevEnvironment(callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *InstallOptions) configureProwInTeamSettings() error {
	if options.Flags.Prow {
		callback := func(env *v1.Environment) error {
			env.Spec.WebHookEngine = v1.WebHookEngineProw
			settings := &env.Spec.TeamSettings
			settings.PromotionEngine = v1.PromotionEngineProw
			log.Info("Configuring the TeamSettings for Prow\n")
			return nil
		}
		err := options.ModifyDevEnvironment(callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *InstallOptions) configureGitOpsMode(configStore configio.ConfigStore, namespace string) (string, string, error) {
	gitOpsDir := ""
	gitOpsEnvDir := ""
	if options.Flags.GitOpsMode {
		// lets disable loading of Secrets from the jx namespace
		options.SkipAuthSecretsMerge = true
		options.Flags.DisableSetKubeContext = true

		var err error
		if options.Flags.Dir == "" {
			options.Flags.Dir, err = os.Getwd()
			if err != nil {
				return "", "", err
			}
		}
		gitOpsDir = filepath.Join(options.Flags.Dir, "jenkins-x-dev-environment")
		gitOpsEnvDir = filepath.Join(gitOpsDir, "env")
		templatesDir := filepath.Join(gitOpsEnvDir, "templates")
		err = os.MkdirAll(templatesDir, util.DefaultWritePermissions)
		if err != nil {
			return "", "", errors.Wrapf(err, "Failed to make GitOps templates directory %s", templatesDir)
		}

		options.modifyDevEnvironmentFn = func(callback func(env *v1.Environment) error) error {
			defaultEnv := kube.CreateDefaultDevEnvironment(namespace)
			_, err := gitOpsModifyEnvironment(templatesDir, kube.LabelValueDevEnvironment, defaultEnv, configStore, callback)
			return err
		}
		options.modifyEnvironmentFn = func(name string, callback func(env *v1.Environment) error) error {
			defaultEnv := &v1.Environment{}
			defaultEnv.Labels = map[string]string{}
			_, err := gitOpsModifyEnvironment(templatesDir, name, defaultEnv, configStore, callback)
			return err
		}
		options.InitOptions.modifyDevEnvironmentFn = options.modifyDevEnvironmentFn
		options.modifyConfigMapCallback = func(name string, callback func(configMap *core_v1.ConfigMap) error) (*core_v1.ConfigMap, error) {
			return gitOpsModifyConfigMap(templatesDir, name, nil, configStore, callback)
		}
		options.modifySecretCallback = func(name string, callback func(secret *core_v1.Secret) error) (*core_v1.Secret, error) {
			if options.Flags.Vault {
				vaultClient, err := options.CreateSystemVaultClient()
				if err != nil {
					return nil, errors.Wrap(err, "retrieving the system vault client")
				}
				vaultConfigStore := configio.NewVaultStore(vaultClient, vault.GitOpsSecretsPath)
				return gitOpsModifySecret(vault.GitOpsTemplatesPath, name, nil, vaultConfigStore, callback)
			}
			return gitOpsModifySecret(templatesDir, name, nil, configStore, callback)
		}
	}

	return gitOpsDir, gitOpsEnvDir, nil
}

func (options *InstallOptions) generateGitOpsDevEnvironmentConfig(gitOpsDir string) error {
	if options.Flags.GitOpsMode {
		log.Infof("\n\nGenerated the source code for the GitOps development environment at %s\n", util.ColorInfo(gitOpsDir))
		log.Infof("You can apply this to the kubernetes cluster at any time in this directory via: %s\n\n", util.ColorInfo("jx step env apply"))

		if !options.Flags.NoGitOpsEnvRepo {
			authConfigSvc, err := options.CreateGitAuthConfigService()
			if err != nil {
				return err
			}
			config := &v1.Environment{
				Spec: v1.EnvironmentSpec{
					Label:             "Development",
					PromotionStrategy: v1.PromotionStrategyTypeNever,
					Kind:              v1.EnvironmentKindTypeDevelopment,
				},
			}
			config.Name = kube.LabelValueDevEnvironment
			var devEnv *v1.Environment
			err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
				devEnv = env
				devEnv.Spec.TeamSettings.UseGitOps = true
				return nil
			})
			if err != nil {
				return err
			}
			envDir, err := util.EnvironmentsDir()
			if err != nil {
				return err
			}
			forkEnvGitURL := ""
			prefix := options.Flags.DefaultEnvironmentPrefix

			git := options.Git()
			gitRepoOptions, err := options.buildGitRepositoryOptionsForEnvironments()
			if err != nil || gitRepoOptions == nil {
				if gitRepoOptions == nil {
					err = errors.New("empty git repository options")
				}
				return errors.Wrap(err, "building the git repository options for environment")
			}
			repo, gitProvider, err := kube.CreateEnvGitRepository(options.BatchMode, authConfigSvc, devEnv, devEnv, config, forkEnvGitURL, envDir,
				gitRepoOptions, options.CreateEnvOptions.HelmValuesConfig, prefix, git, options.In, options.Out, options.Err)
			if err != nil {
				return errors.Wrapf(err, "failed to create git repository for the dev Environment source")
			}
			dir := gitOpsDir
			err = git.Init(dir)
			if err != nil {
				return err
			}
			err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
				env.Spec.Source.URL = repo.CloneURL
				env.Spec.Source.Ref = "master"
				return nil
			})
			if err != nil {
				return err
			}

			err = git.Add(dir, ".gitignore")
			if err != nil {
				return err
			}
			err = git.Add(dir, "*")
			if err != nil {
				return err
			}
			err = options.Git().CommitIfChanges(dir, "Initial import of Dev Environment source")
			if err != nil {
				return err
			}
			userAuth := gitProvider.UserAuth()
			pushGitURL, err := git.CreatePushURL(repo.CloneURL, &userAuth)
			if err != nil {
				return err
			}
			err = git.SetRemoteURL(dir, "origin", pushGitURL)
			if err != nil {
				return err
			}
			err = git.PushMaster(dir)
			if err != nil {
				return err
			}
			log.Infof("Pushed Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
		}
	}

	return nil
}

func (options *InstallOptions) applyGitOpsDevEnvironmentConfig(gitOpsEnvDir string, namespace string) error {
	if options.Flags.GitOpsMode && !options.Flags.NoGitOpsEnvApply {
		applyEnv := true
		if !options.BatchMode {
			if !util.Confirm("Would you like to setup the Development Environment from the source code now?", true, "Do you want to apply the development environment helm charts now?", options.In, options.Out, options.Err) {
				applyEnv = false
			}
		}

		if applyEnv {
			envApplyOptions := &StepEnvApplyOptions{
				StepEnvOptions: StepEnvOptions{
					StepOptions: StepOptions{
						CommonOptions: options.CommonOptions,
					},
				},
				Dir:       gitOpsEnvDir,
				Namespace: namespace,
			}
			err := envApplyOptions.Run()
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func (options *InstallOptions) installHelmBinaries() error {
	initOpts := &options.InitOptions
	helmBinary := initOpts.HelmBinary()
	dependencies := []string{}
	if !initOpts.Flags.RemoteTiller && !initOpts.Flags.NoTiller {
		binDir, err := util.JXBinLocation()
		if err != nil {
			return errors.Wrap(err, "reading jx bin location")
		}
		_, install, err := shouldInstallBinary("tiller")
		if !install && err == nil {
			confirm := &survey.Confirm{
				Message: "Uninstalling existing tiller binary:",
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
				Message: "Uninstalling existing helm binary:",
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
		options.Helm().SetHost(tillerAddress())
	}
	dependencies = append(dependencies, helmBinary)
	return options.installMissingDependencies(dependencies)
}

func (options *InstallOptions) configureCloudProivderPreInit(client kubernetes.Interface) error {
	switch options.Flags.Provider {
	case AKS:
		err := options.createClusterAdmin()
		if err != nil {
			return errors.Wrap(err, "creating cluster admin for AKS cloud provider")
		}
		log.Success("created role cluster-admin")
	case AWS:
		fallthrough
	case EKS:
		err := options.ensureDefaultStorageClass(client, "gp2", "kubernetes.io/aws-ebs", "gp2")
		if err != nil {
			return errors.Wrap(err, "ensuring default storage for EKS/AWS cloud provider")
		}
	case MINIKUBE:
		if options.Flags.Domain == "" {
			ip, err := options.getCommandOutput("", "minikube", "ip")
			if err != nil {
				return errors.Wrap(err, "failed to get the IP from Minikube")
			}
			options.Flags.Domain = ip + ".nip.io"
		}
	default:
		return nil
	}
	return nil
}

func (options *InstallOptions) configureCloudProivderPostInit(client kubernetes.Interface, namespace string) error {
	switch options.Flags.Provider {
	case MINISHIFT:
		fallthrough
	case OPENSHIFT:
		err := options.enableOpenShiftSCC(namespace)
		if err != nil {
			return errors.Wrap(err, "failed to enable the OpenShiftSCC")
		}
	case IKS:
		err := options.addHelmBinaryRepoIfMissing(DEFAULT_IBMREPO_URL, "ibm", "", "")
		if err != nil {
			return errors.Wrap(err, "failed to add the IBM helm repo")
		}
		err = options.Helm().UpdateRepo()
		if err != nil {
			return errors.Wrap(err, "failed to update the helm repo")
		}
		err = options.Helm().UpgradeChart("ibm/ibmcloud-block-storage-plugin", "ibmcloud-block-storage-plugin",
			"default", nil, true, nil, false, false, nil, nil, "", "", "")
		if err != nil {
			return errors.Wrap(err, "failed to install/upgrade the IBM Cloud Block Storage drivers")
		}
		return options.changeDefaultStorageClass(client, "ibmc-block-bronze")
	default:
		return nil
	}

	return nil
}

func (options *InstallOptions) configureDockerRegistry(client kubernetes.Interface, namespace string) error {
	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	dockerRegistryConfig, dockerRegistry, err := options.configureCloudProviderRegistry(client, namespace)
	if err != nil {
		return errors.Wrap(err, "configure cloud provider docker registry")
	}
	if dockerRegistryConfig != "" {
		helmConfig.PipelineSecrets.DockerConfig = dockerRegistryConfig
	}
	if dockerRegistry != "" {
		if helmConfig.Jenkins.Servers.Global.EnvVars == nil {
			helmConfig.Jenkins.Servers.Global.EnvVars = map[string]string{}
		}
		helmConfig.Jenkins.Servers.Global.EnvVars["DOCKER_REGISTRY"] = dockerRegistry
	}
	return nil
}

func (options *InstallOptions) configureCloudProviderRegistry(client kubernetes.Interface, namespace string) (string, string, error) {
	dockerRegistry, err := options.dockerRegistryValue()
	if err != nil {
		return "", "", err
	}
	kubeConfig, _, err := options.Kube().LoadConfig()
	if err != nil {
		return "", "", err
	}
	switch options.Flags.Provider {
	case AKS:
		server := kube.CurrentServer(kubeConfig)
		azureCLI := aks.NewAzureRunner()
		resourceGroup, name, cluster, err := azureCLI.GetClusterClient(server)
		if err != nil {
			return "", "", errors.Wrap(err, "getting cluster from Azure")
		}
		registryID := ""
		config, dockerRegistry, registryID, err := azureCLI.GetRegistry(resourceGroup, name, dockerRegistry)
		if err != nil {
			return "", "", errors.Wrap(err, "getting registry configuration from Azure")
		}
		azureCLI.AssignRole(cluster, registryID)
		log.Infof("Assign AKS %s a reader role for ACR %s\n", util.ColorInfo(server), util.ColorInfo(dockerRegistry))
		return config, dockerRegistry, nil
	case IKS:
		dockerRegistry = iks.GetClusterRegistry(client)
		config, err := iks.GetRegistryConfigJSON(dockerRegistry)
		if err != nil {
			return "", "", errors.Wrap(err, "getting IKS registry configuration")
		}
		return config, dockerRegistry, nil
	case MINISHIFT:
		fallthrough
	case OPENSHIFT:
		if dockerRegistry == "docker-registry.default.svc:5000" {
			config, err := options.enableOpenShiftRegistryPermissions(namespace, dockerRegistry)
			if err != nil {
				return "", "", errors.Wrap(err, "enabling OpenShift registry permissions")
			}
			return config, dockerRegistry, nil
		}
	}

	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	return helmConfig.PipelineSecrets.DockerConfig, dockerRegistry, nil
}

func (options *InstallOptions) setMinikubeFromContext() error {
	currentContext := ""
	var err error
	if !options.Flags.DisableSetKubeContext {
		currentContext, err = options.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return errors.Wrap(err, "failed to get the current context")
		}
	}
	if currentContext == "minikube" {
		if options.Flags.Provider == "" {
			options.Flags.Provider = MINIKUBE
		}
	}
	return nil
}

func (options *InstallOptions) registerAllCRDs() error {
	if !options.GitOpsMode {
		apisClient, err := options.ApiExtensionsClient()
		if err != nil {
			return errors.Wrap(err, "failed to create the API extensions client")
		}
		kube.RegisterAllCRDs(apisClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *InstallOptions) installCloudProviderDependencies() error {
	dependencies := []string{}
	err := options.installRequirements(options.Flags.Provider, dependencies...)
	if err != nil {
		return errors.Wrap(err, "installing cloud provider dependencies")
	}
	return nil
}

func (options *InstallOptions) getAdminSecrets(configStore configio.ConfigStore, providerEnvDir string, cloudEnvironmentSecretsLocation string) (string, *config.AdminSecretsConfig, error) {
	cloudEnvironmentSopsLocation := filepath.Join(providerEnvDir, CloudEnvSopsConfigFile)
	if _, err := os.Stat(providerEnvDir); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("cloud environment dir %s not found", providerEnvDir)
	}
	sopsFileExists, err := util.FileExists(cloudEnvironmentSopsLocation)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to look for "+cloudEnvironmentSopsLocation)
	}

	adminSecretsServiceInit := false

	if sopsFileExists {
		log.Infof("Attempting to decrypt secrets file %s\n", util.ColorInfo(cloudEnvironmentSecretsLocation))
		// need to decrypt secrets now
		err = options.Helm().DecryptSecrets(cloudEnvironmentSecretsLocation)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to decrypt "+cloudEnvironmentSecretsLocation)
		}

		cloudEnvironmentSecretsDecryptedLocation := filepath.Join(providerEnvDir, CloudEnvSecretsFile+".dec")
		decryptedSecretsFile, err := util.FileExists(cloudEnvironmentSecretsDecryptedLocation)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to look for "+cloudEnvironmentSecretsDecryptedLocation)
		}

		if decryptedSecretsFile {
			log.Infof("Successfully decrypted %s\n", util.ColorInfo(cloudEnvironmentSecretsDecryptedLocation))
			cloudEnvironmentSecretsLocation = cloudEnvironmentSecretsDecryptedLocation

			err = options.AdminSecretsService.NewAdminSecretsConfigFromSecret(cloudEnvironmentSecretsDecryptedLocation)
			if err != nil {
				return "", nil, errors.Wrap(err, "failed to create the admin secret config service from the decrypted secrets file")
			}
			adminSecretsServiceInit = true
		}
	}

	if !adminSecretsServiceInit {
		err = options.AdminSecretsService.NewAdminSecretsConfig()
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to create the admin secret config service")
		}
	}

	dir, err := util.ConfigDir()
	if err != nil {
		return "", nil, errors.Wrap(err, "creating a temporary config dir for Git credentials")
	}

	adminSecrets := &options.AdminSecretsService.Secrets
	adminSecretsFileName := filepath.Join(dir, AdminSecretsFile)
	err = configStore.WriteObject(adminSecretsFileName, adminSecrets)
	if err != nil {
		return "", nil, errors.Wrapf(err, "writing the admin secrets in the secrets file '%s'", adminSecretsFileName)
	}

	if options.Flags.Vault {
		err := options.storeAdminCredentialsInVault(&options.AdminSecretsService)
		if err != nil {
			return "", nil, errors.Wrapf(err, "storing the admin credentials in vault")
		}
	}

	return adminSecretsFileName, adminSecrets, nil
}

func (options *InstallOptions) createSystemVault(client kubernetes.Interface, namespace string, ic *kube.IngressConfig) error {
	if options.Flags.GitOpsMode && !options.Flags.NoGitOpsVault || options.Flags.Vault {
		if options.Flags.Provider != GKE {
			return fmt.Errorf("system vault is not supported for %s provider", options.Flags.Provider)
		}

		// Configure the vault flag if only GitOps mode is on
		options.Flags.Vault = true

		err := InstallVaultOperator(&options.CommonOptions, namespace)
		if err != nil {
			return err
		}

		// Create a new System vault
		cvo := &CreateVaultOptions{
			CreateOptions: CreateOptions{
				CommonOptions: options.CommonOptions,
			},
			UpgradeIngressOptions: UpgradeIngressOptions{
				CreateOptions: CreateOptions{
					CommonOptions: options.CommonOptions,
				},
				IngressConfig: *ic,
			},
			Namespace: namespace,
		}
		vaultOperatorClient, err := cvo.CreateVaultOperatorClient()
		if err != nil {
			return err
		}

		if kubevault.FindVault(vaultOperatorClient, vault.SystemVaultName, namespace) {
			log.Infof("System vault named %s in namespace %s already exists\n",
				util.ColorInfo(vault.SystemVaultName), util.ColorInfo(namespace))
		} else {
			log.Info("Creating new system vault\n")
			err = cvo.createVault(vaultOperatorClient, vault.SystemVaultName)
			if err != nil {
				return err
			}
			log.Infof("System vault created named %s in namespace %s.\n",
				util.ColorInfo(vault.SystemVaultName), util.ColorInfo(namespace))
		}
		err = secrets.NewSecretLocation(client, namespace).SetInVault(options.Flags.Vault)
		if err != nil {
			return errors.Wrap(err, "configuring secrets location")
		}
	}
	return nil
}

func (options *InstallOptions) storeSecretYamlFilesInVault(path string, files ...string) error {
	vaultClient, err := options.CreateSystemVaultClient()
	if err != nil {
		return errors.Wrap(err, "retrieving the system vault client")
	}

	err = vault.WriteYamlFiles(vaultClient, path, files...)
	if err != nil {
		return errors.Wrapf(err, "storing in vault the secret YAML files: %s", strings.Join(files, ","))
	}

	return nil
}

func (options *InstallOptions) storeAdminCredentialsInVault(svc *config.AdminSecretsService) error {
	vaultClient, err := options.CreateSystemVaultClient()
	if err != nil {
		return errors.Wrap(err, "retrieving the system vault client")
	}
	secrets := map[vault.AdminSecret]config.BasicAuth{
		vault.JenkinsAdminSecret:     svc.JenkinsAuth(),
		vault.IngressAdminSecret:     svc.IngressAuth(),
		vault.ChartmuseumAdminSecret: svc.ChartMuseumAuth(),
		vault.GrafanaAdminSecret:     svc.GrafanaAuth(),
		vault.NexusAdminSecret:       svc.NexusAuth(),
	}
	for secretName, secret := range secrets {
		path := vault.AdminSecretPath(secretName)
		err := vault.WriteBasicAuth(vaultClient, path, secret)
		if err != nil {
			return errors.Wrapf(err, "storing in vault the basic auth credentials for %s", secretName)
		}
	}
	return nil
}

func (options *InstallOptions) configureBuildPackMode() error {
	ebp := &EditBuildPackOptions{
		BuildPackName: options.Flags.BuildPackName,
	}
	ebp.CommonOptions = options.CommonOptions

	return ebp.Run()
}

func (options *InstallOptions) saveIngressConfig() (*kube.IngressConfig, error) {
	helmConfig := &options.CreateEnvOptions.HelmValuesConfig
	domain := helmConfig.ExposeController.Config.Domain
	if domain == "" {
		domain = options.InitOptions.Flags.Domain
		helmConfig.ExposeController.Config.Domain = domain
	}

	exposeController := options.CreateEnvOptions.HelmValuesConfig.ExposeController
	tls, err := util.ParseBool(exposeController.Config.TLSAcme)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TLS exposecontroller boolean %v", err)
	}
	ic := kube.IngressConfig{
		Domain:  domain,
		TLS:     tls,
		Exposer: exposeController.Config.Exposer,
	}
	// save ingress config details to a configmap
	_, err = options.saveAsConfigMap(kube.IngressConfigConfigmap, ic)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

func (options *InstallOptions) saveClusterConfig() error {
	if !options.Flags.DisableSetKubeContext {
		var jxInstallConfig *kube.JXInstallConfig
		kubeConfig, _, err := options.Kube().LoadConfig()
		if err != nil {
			return errors.Wrap(err, "retrieving the current kube config")
		}
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

		_, err = options.ModifyConfigMap(kube.ConfigMapNameJXInstallConfig, func(cm *core_v1.ConfigMap) error {
			data := util.ToStringMapStringFromStruct(jxInstallConfig)
			cm.Data = data
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *InstallOptions) configureJenkins(namespace string) error {
	if !options.Flags.GitOpsMode {
		if !options.Flags.Prow {
			log.Info("Getting Jenkins API Token\n")
			if isOpenShiftProvider(options.Flags.Provider) {
				options.CreateJenkinsUserOptions.CommonOptions = options.CommonOptions
				options.CreateJenkinsUserOptions.Password = options.AdminSecretsService.Flags.DefaultAdminPassword
				options.CreateJenkinsUserOptions.Username = "jenkins-admin"
				jenkinsSaToken, err := options.getCommandOutput("", "oc", "serviceaccounts", "get-token", "jenkins", "-n", namespace)
				if err != nil {
					return err
				}
				options.CreateJenkinsUserOptions.BearerToken = jenkinsSaToken
				options.CreateJenkinsUserOptions.Run()
			} else {
				// Wait for Jenkins service to be ready after installation before trying to generate the token
				time.Sleep(2 * time.Second)
				err := options.retry(3, 2*time.Second, func() (err error) {
					options.CreateJenkinsUserOptions.CommonOptions = options.CommonOptions
					options.CreateJenkinsUserOptions.Password = options.AdminSecretsService.Flags.DefaultAdminPassword
					options.CreateJenkinsUserOptions.UseBrowser = true
					if options.BatchMode {
						options.CreateJenkinsUserOptions.BatchMode = true
						options.CreateJenkinsUserOptions.Headless = true
						log.Info("Attempting to find the Jenkins API Token with the browser in headless mode...\n")
					}
					err = options.CreateJenkinsUserOptions.Run()
					return
				})
				if err != nil {
					return errors.Wrap(err, "failed to get the Jenkins API token")
				}
			}
		}

		if !options.Flags.Prow {
			err := options.updateJenkinsURL([]string{namespace})
			if err != nil {
				log.Warnf("failed to update the Jenkins external URL")
			}
		}
	}
	return nil
}

func (options *InstallOptions) installAddons() error {
	if !options.Flags.GitOpsMode {
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
	}
	return nil
}

func (options *InstallOptions) createEnvironments(namespace string) error {
	if options.Flags.DefaultEnvironmentPrefix == "" {
		options.Flags.DefaultEnvironmentPrefix = strings.ToLower(randomdata.SillyName())
	}

	if !options.Flags.NoDefaultEnvironments {
		createEnvironments := true
		if options.Flags.GitOpsMode {
			options.SetDevNamespace(namespace)
			options.CreateEnvOptions.CommonOptions = options.CommonOptions
			options.CreateEnvOptions.GitOpsMode = true
			options.CreateEnvOptions.modifyDevEnvironmentFn = options.modifyDevEnvironmentFn
			options.CreateEnvOptions.modifyEnvironmentFn = options.modifyEnvironmentFn
		} else {
			createEnvironments = false

			jxClient, _, err := options.JXClient()
			if err != nil {
				return errors.Wrap(err, "failed to create the jx client")
			}

			// lets only recreate the environments if its the first time we run this
			_, envNames, err := kube.GetEnvironments(jxClient, namespace)
			if err != nil || len(envNames) <= 1 {
				createEnvironments = true
			}

		}
		if createEnvironments {
			log.Info("Creating default staging and production environments\n")
			gitRepoOptions, err := options.buildGitRepositoryOptionsForEnvironments()
			if err != nil || gitRepoOptions == nil {
				return errors.Wrap(err, "building the Git repository options for environments")
			}
			options.CreateEnvOptions.GitRepositoryOptions = *gitRepoOptions

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
	return nil
}

func (options *InstallOptions) modifySecrets(helmConfig *config.HelmValuesConfig, adminSecrets *config.AdminSecretsConfig) error {
	var err error
	data := make(map[string][]byte)
	data[ExtraValuesFile], err = yaml.Marshal(helmConfig)
	if err != nil {
		return err
	}
	data[AdminSecretsFile], err = yaml.Marshal(adminSecrets)
	if err != nil {
		return err
	}
	_, err = options.ModifySecret(JXInstallConfig, func(secret *core_v1.Secret) error {
		secret.Data = data
		return nil
	})
	return err
}

// ModifySecret modifies the Secret either live or via the file system if generating the GitOps source
func (options *InstallOptions) ModifySecret(name string, callback func(*core_v1.Secret) error) (*core_v1.Secret, error) {
	if options.modifySecretCallback == nil {
		options.modifySecretCallback = func(name string, callback func(*core_v1.Secret) error) (*core_v1.Secret, error) {
			kubeClient, ns, err := options.KubeClientAndDevNamespace()
			if err != nil {
				return nil, err
			}
			return kube.DefaultModifySecret(kubeClient, ns, name, callback, nil)
		}
	}
	return options.modifySecretCallback(name, callback)
}

// ModifyConfigMap modifies the ConfigMap either live or via the file system if generating the GitOps source
func (options *InstallOptions) ModifyConfigMap(name string, callback func(*core_v1.ConfigMap) error) (*core_v1.ConfigMap, error) {
	if options.modifyConfigMapCallback == nil {
		options.modifyConfigMapCallback = func(name string, callback func(*core_v1.ConfigMap) error) (*core_v1.ConfigMap, error) {
			kubeClient, ns, err := options.KubeClientAndDevNamespace()
			if err != nil {
				return nil, err
			}
			return kube.DefaultModifyConfigMap(kubeClient, ns, name, callback, nil)
		}
	}
	return options.modifyConfigMapCallback(name, callback)
}

// gitOpsModifyConfigMap provides a helper function to lazily create, modify and save the YAML file in the given directory
func gitOpsModifyConfigMap(dir string, name string, defaultResource *core_v1.ConfigMap, configStore configio.ConfigStore,
	callback func(configMap *core_v1.ConfigMap) error) (*core_v1.ConfigMap, error) {
	answer := core_v1.ConfigMap{}
	fileName := filepath.Join(dir, name+"-configmap.yaml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not check if file exists %s", fileName)
	}
	if exists {
		err = configStore.ReadObject(fileName, &answer)
		if err != nil {
			return &answer, errors.Wrapf(err, "Failed to unmarshall YAML file %s", fileName)
		}
	} else if defaultResource != nil {
		answer = *defaultResource
	} else {
		answer.Name = name
	}
	err = callback(&answer)
	if err != nil {
		return &answer, err
	}
	if answer.APIVersion == "" {
		answer.APIVersion = "v1"
	}
	if answer.Kind == "" {
		answer.Kind = "ConfigMap"
	}
	err = configStore.WriteObject(fileName, &answer)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not save file %s", fileName)
	}
	return &answer, nil
}

// gitOpsModifySecret provides a helper function to lazily create, modify and save the YAML file in the given directory
func gitOpsModifySecret(dir string, name string, defaultResource *core_v1.Secret, configStore configio.ConfigStore,
	callback func(secret *core_v1.Secret) error) (*core_v1.Secret, error) {
	answer := core_v1.Secret{}
	fileName := filepath.Join(dir, name+"-secret.yaml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not check if file exists %s", fileName)
	}
	if exists {
		// lets unmarshall the data
		err = configStore.ReadObject(fileName, &answer)
		if err != nil {
			return &answer, err
		}
	} else if defaultResource != nil {
		answer = *defaultResource
	} else {
		answer.Name = name
	}
	err = callback(&answer)
	if err != nil {
		return &answer, err
	}
	if answer.APIVersion == "" {
		answer.APIVersion = "v1"
	}
	if answer.Kind == "" {
		answer.Kind = "Secret"
	}
	err = configStore.WriteObject(fileName, &answer)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not save file %s", fileName)
	}
	return &answer, nil
}

// gitOpsModifyEnvironment provides a helper function to lazily create, modify and save the YAML file in the given directory
func gitOpsModifyEnvironment(dir string, name string, defaultEnvironment *v1.Environment, configStore configio.ConfigStore,
	callback func(*v1.Environment) error) (*v1.Environment, error) {
	answer := v1.Environment{}
	fileName := filepath.Join(dir, name+"-env.yaml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not check if file exists %s", fileName)
	}
	if exists {
		// lets unmarshal the data
		err := configStore.ReadObject(fileName, &answer)
		if err != nil {
			return &answer, err
		}
	} else if defaultEnvironment != nil {
		answer = *defaultEnvironment
	}
	err = callback(&answer)
	if err != nil {
		return &answer, err
	}
	answer.Name = name
	if answer.APIVersion == "" {
		answer.APIVersion = jenkinsio.GroupAndVersion
	}
	if answer.Kind == "" {
		answer.Kind = "Environment"
	}
	err = configStore.WriteObject(fileName, &answer)
	if err != nil {
		return &answer, errors.Wrapf(err, "Could not save file %s", fileName)
	}
	return &answer, nil
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

func (options *InstallOptions) enableOpenShiftRegistryPermissions(ns string, dockerRegistry string) (string, error) {
	log.Infof("Enabling permissions for OpenShift registry in namespace %s\n", ns)
	// Open the registry so any authenticated user can pull images from the jx namespace
	err := options.RunCommand("oc", "adm", "policy", "add-role-to-group", "system:image-puller", "system:authenticated", "-n", ns)
	if err != nil {
		return "", err
	}
	err = options.ensureServiceAccount(ns, "jenkins-x-registry")
	if err != nil {
		return "", err
	}
	err = options.RunCommand("oc", "adm", "policy", "add-cluster-role-to-user", "registry-admin", "system:serviceaccount:"+ns+":jenkins-x-registry")
	if err != nil {
		return "", err
	}
	registryToken, err := options.getCommandOutput("", "oc", "serviceaccounts", "get-token", "jenkins-x-registry", "-n", ns)
	if err != nil {
		return "", err
	}
	return `{"auths": {"` + dockerRegistry + `": {"auth": "` + base64.StdEncoding.EncodeToString([]byte("serviceaccount:"+registryToken)) + `"}}}`, nil
}

func (options *InstallOptions) logAdminPassword() {
	astrix := `

	********************************************************

	     NOTE: %s

	********************************************************

	`
	if options.Flags.Vault {
		log.Infof(astrix+"\n", fmt.Sprintf("Your admin password is in vault: %s", util.ColorInfo("eval `jx get vault-config` && vault kv get secret/admin/jenkins")))
	} else {
		log.Infof(astrix+"\n", fmt.Sprintf("Your admin password is: %s", util.ColorInfo(options.AdminSecretsService.Flags.DefaultAdminPassword)))
	}
}

// LoadVersionFromCloudEnvironmentsDir loads a version from the cloud environments directory
func LoadVersionFromCloudEnvironmentsDir(wrkDir string, configStore configio.ConfigStore) (string, error) {
	version := ""
	path := filepath.Join(wrkDir, "Makefile")
	exists, err := util.FileExists(path)
	if err != nil {
		return version, err
	}
	if !exists {
		return version, fmt.Errorf("File does not exist %s", path)
	}
	data, err := configStore.Read(path)
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

	options.Debugf("Current configuration dir: %s\n", configDir)
	options.Debugf("options.Flags.CloudEnvRepository: %s\n", options.Flags.CloudEnvRepository)
	options.Debugf("options.Flags.LocalCloudEnvironment: %t\n", options.Flags.LocalCloudEnvironment)

	if options.Flags.LocalCloudEnvironment {
		currentDir, err := os.Getwd()
		if err != nil {
			return wrkDir, fmt.Errorf("error getting current working directory %v", err)
		}
		log.Infof("Copying local dir %s to %s\n", currentDir, wrkDir)

		return wrkDir, util.CopyDir(currentDir, wrkDir, true)
	}
	if options.Flags.CloudEnvRepository == "" {
		options.Flags.CloudEnvRepository = DefaultCloudEnvironmentsURL
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

func (options *InstallOptions) getPipelineGitAuth() (*auth.AuthServer, *auth.UserAuth, error) {
	authConfigSvc, err := options.CreateGitAuthConfigService()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create the git auth config service")
	}
	authConfig := authConfigSvc.Config()
	if authConfig == nil {
		return nil, nil, errors.New("empty Git config")
	}
	server, user := authConfig.GetPipelineAuth()
	return server, user, nil
}

func (options *InstallOptions) waitForInstallToBeReady(ns string) error {
	client, err := options.KubeClient()
	if err != nil {
		return err
	}

	log.Warnf("waiting for install to be ready, if this is the first time then it will take a while to download images\n")

	return kube.WaitForAllDeploymentsToBeReady(client, ns, 30*time.Minute)

}

func (options *InstallOptions) saveChartmuseumAuthConfig() error {

	authConfigSvc, err := options.CreateChartmuseumAuthConfigService()
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

func (options *InstallOptions) addGitServersToJenkinsConfig(helmConfig *config.HelmValuesConfig) error {
	gitAuthCfg, err := options.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create the git auth config service")
	}
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

func (options *InstallOptions) saveAsConfigMap(name string, config interface{}) (*core_v1.ConfigMap, error) {
	return options.ModifyConfigMap(name, func(cm *core_v1.ConfigMap) error {
		data := util.ToStringMapStringFromStruct(config)
		cm.Data = data
		return nil
	})
}
