package verify

import (
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/gits"

	"github.com/jenkins-x/jx/pkg/boot"

	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/factory"
	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StepVerifyPreInstallOptions contains the command line flags
type StepVerifyPreInstallOptions struct {
	StepVerifyOptions
	Debug                bool
	Dir                  string
	LazyCreate           bool
	DisableVerifyHelm    bool
	LazyCreateFlag       string
	Namespace            string
	TestKanikoSecretData string
	WorkloadIdentity     bool
}

// NewCmdStepVerifyPreInstall creates the `jx step verify pod` command
func NewCmdStepVerifyPreInstall(commonOpts *opts.CommonOptions) *cobra.Command {

	options := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "preinstall",
		Aliases: []string{"pre-install", "pre"},
		Short:   "Verifies all of the cloud infrastructure is setup before we try to boot up a cluster via 'jx boot'",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Debug, "debug", "", false, "Output logs of any failed pod")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the install requirements file")
	cmd.Flags().StringVarP(&options.LazyCreateFlag, "lazy-create", "", "", fmt.Sprintf("Specify true/false as to whether to lazily create missing resources. If not specified it is enabled if Terraform is not specified in the %s file", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "the namespace that Jenkins X will be booted into. If not specified it defaults to $DEPLOY_NAMESPACE")
	cmd.Flags().BoolVarP(&options.WorkloadIdentity, "workload-identity", "", false, "Enable this if using GKE Workload Identity to avoid reconnecting to the Cluster.")

	return cmd
}

// Run implements this command
func (o *StepVerifyPreInstallOptions) Run() error {
	info := util.ColorInfo
	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}

	requirements, err = o.gatherRequirements(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	o.LazyCreate, err = requirements.IsLazyCreateSecrets(o.LazyCreateFlag)
	if err != nil {
		return err
	}

	// lets find the namespace to use
	ns, err := o.GetDeployNamespace(o.Namespace)
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	err = o.verifyTLS(requirements)
	if err != nil {
		return errors.WithStack(err)
	}

	o.SetDevNamespace(ns)

	log.Logger().Infof("verifying the kubernetes cluster before we try to boot Jenkins X in namespace: %s", info(ns))
	if o.LazyCreate {
		log.Logger().Infof("we will try to lazily create any missing resources to get the current cluster ready to boot Jenkins X")
	} else {
		log.Logger().Warn("lazy create of cloud resources is disabled")

	}

	err = o.verifyDevNamespace(kubeClient, ns)
	if err != nil {
		if o.LazyCreate {
			log.Logger().Infof("attempting to lazily create the deploy namespace %s", info(ns))

			err = kube.EnsureDevNamespaceCreatedWithoutEnvironment(kubeClient, ns)
			if err != nil {
				return errors.Wrapf(err, "failed to lazily create the namespace %s", ns)
			}
			// lets rerun the verify step to ensure its all sorted now
			err = o.verifyDevNamespace(kubeClient, ns)
			if err != nil {
				return errors.Wrapf(err, "failed to verify the namespace %s", ns)
			}
		}
	}

	err = o.verifyIngress(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	no := &namespace.NamespaceOptions{}
	no.CommonOptions = o.CommonOptions
	no.Args = []string{ns}
	log.Logger().Infof("setting the local kubernetes context to the deploy namespace %s", info(ns))
	err = no.Run()
	if err != nil {
		return err
	}

	po := &StepVerifyPackagesOptions{}
	po.CommonOptions = o.CommonOptions
	err = po.Run()
	if err != nil {
		return err
	}

	err = o.VerifyInstallConfig(kubeClient, ns, requirements, requirementsFileName)
	if err != nil {
		return err
	}

	err = o.verifyStorage(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	if !o.DisableVerifyHelm {
		err = o.verifyHelm(ns)
		if err != nil {
			return err
		}
	}

	if requirements.Kaniko {
		if requirements.Cluster.Provider == cloud.GKE {
			log.Logger().Infof("validating the kaniko secret in namespace %s", info(ns))

			err = o.validateKaniko(ns)
			if err != nil {
				if o.LazyCreate {
					log.Logger().Infof("attempting to lazily create the deploy namespace %s", info(ns))

					err = o.lazyCreateKanikoSecret(requirements, ns)
					if err != nil {
						return errors.Wrapf(err, "failed to lazily create the kaniko secret in: %s", ns)
					}
					// lets rerun the verify step to ensure its all sorted now
					err = o.validateKaniko(ns)
				}
			}
			if err != nil {
				return err
			}
		}
	}

	if requirements.Webhook == config.WebhookTypeLighthouse {
		// we don't need the ConfigMaps for prow yet
		err = o.verifyProwConfigMaps(kubeClient, ns)
		if err != nil {
			return err
		}
	}

	log.Logger().Infof("the cluster looks good, you are ready to '%s' now!", info("jx boot"))
	fmt.Println()
	return nil
}

// EnsureHelm ensures helm is installed
func (o *StepVerifyPreInstallOptions) verifyHelm(ns string) error {
	log.Logger().Debug("Verifying Helm...")
	// lets make sure we don't try use tiller
	o.EnableRemoteKubeCluster()
	_, err := o.Helm().Version(false)
	if err != nil {
		err = o.InstallHelm()
		if err != nil {
			return errors.Wrap(err, "failed to install Helm")
		}
	}
	cfg := opts.InitHelmConfig{
		Namespace:       ns,
		OnlyHelmClient:  true,
		Helm3:           false,
		SkipTiller:      true,
		GlobalTiller:    false,
		TillerNamespace: "",
		TillerRole:      "",
	}
	err = o.InitHelm(cfg)
	if err != nil {
		return errors.Wrapf(err, "initializing helm with config: %v", cfg)
	}
	log.Logger().Infof("helm client is setup")

	o.EnableRemoteKubeCluster()
	_, err = o.AddHelmBinaryRepoIfMissing(kube.DefaultChartMuseumURL, kube.DefaultChartMuseumJxRepoName, "", "")
	if err != nil {
		return errors.Wrapf(err, "adding '%s' helm charts repository", kube.DefaultChartMuseumURL)
	}
	log.Logger().Infof("ensure we have the helm repository %s", kube.DefaultChartMuseumURL)

	return nil
}

func (o *StepVerifyPreInstallOptions) verifyDevNamespace(kubeClient kubernetes.Interface, ns string) error {
	log.Logger().Debug("Verifying Dev Namespace...")
	ns, envName, err := kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return err
	}
	if ns == "" {
		return fmt.Errorf("No dev namespace name found")
	}
	if envName == "" {
		return fmt.Errorf("Namespace %s has no team label", ns)
	}
	return nil
}

func (o *StepVerifyPreInstallOptions) lazyCreateKanikoSecret(requirements *config.RequirementsConfig, ns string) error {
	log.Logger().Debugf("Lazily creating the kaniko secret")
	io := &create.InstallOptions{}
	io.CommonOptions = o.CommonOptions
	io.Flags.Kaniko = true
	io.Flags.Namespace = ns
	io.Flags.Provider = requirements.Cluster.Provider
	io.SetInstallValues(map[string]string{
		kube.ClusterName: requirements.Cluster.ClusterName,
		kube.ProjectID:   requirements.Cluster.ProjectID,
	})
	if o.TestKanikoSecretData != "" {
		io.AdminSecretsService.Flags.KanikoSecret = o.TestKanikoSecretData
	} else {
		err := io.ConfigureKaniko()
		if err != nil {
			return err
		}
	}
	data := io.AdminSecretsService.Flags.KanikoSecret
	if data == "" {
		return fmt.Errorf("failed to create the kaniko secret data")
	}
	return o.createKanikoSecret(ns, data)
}

// VerifyInstallConfig lets ensure we modify the install ConfigMap with the requirements
func (o *StepVerifyPreInstallOptions) VerifyInstallConfig(kubeClient kubernetes.Interface, ns string, requirements *config.RequirementsConfig, requirementsFileName string) error {
	log.Logger().Debug("Verifying Install Config...")
	_, err := kube.DefaultModifyConfigMap(kubeClient, ns, kube.ConfigMapNameJXInstallConfig,
		func(configMap *corev1.ConfigMap) error {
			secretsLocation := string(secrets.FileSystemLocationKind)
			if requirements.SecretStorage == config.SecretStorageTypeVault {
				secretsLocation = string(secrets.VaultLocationKind)
			}
			modifyMapIfNotBlank(configMap.Data, kube.KubeProvider, requirements.Cluster.Provider)
			modifyMapIfNotBlank(configMap.Data, kube.ProjectID, requirements.Cluster.ProjectID)
			modifyMapIfNotBlank(configMap.Data, kube.ClusterName, requirements.Cluster.ClusterName)
			modifyMapIfNotBlank(configMap.Data, secrets.SecretsLocationKey, secretsLocation)
			return nil
		}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving secrets location in ConfigMap %s in namespace %s", kube.ConfigMapNameJXInstallConfig, ns)
	}
	return nil
}

// gatherRequirements gathers cluster requirements and connects to the cluster if required
func (o *StepVerifyPreInstallOptions) gatherRequirements(requirements *config.RequirementsConfig, requirementsFileName string) (*config.RequirementsConfig, error) {
	log.Logger().Debug("Gathering Requirements...")
	if o.BatchMode {
		isTerraform := os.Getenv(config.RequirementTerraform)
		if isTerraform == "true" {
			requirements.Terraform = true
			if "" != os.Getenv(config.RequirementClusterName) {
				requirements.Cluster.ClusterName = os.Getenv(config.RequirementClusterName)
			}
			if "" != os.Getenv(config.RequirementProject) {
				requirements.Cluster.ProjectID = os.Getenv(config.RequirementProject)
			}
			if "" != os.Getenv(config.RequirementZone) {
				requirements.Cluster.Zone = os.Getenv(config.RequirementZone)
			}
			if "" != os.Getenv(config.RequirementEnvGitOwner) {
				requirements.Cluster.EnvironmentGitOwner = os.Getenv(config.RequirementEnvGitOwner)
			}
			if "" != os.Getenv(config.RequirementExternalDNSServiceAccountName) {
				requirements.Cluster.ExternalDNSSAName = os.Getenv(config.RequirementExternalDNSServiceAccountName)
			}
			if "" != os.Getenv(config.RequirementVaultServiceAccountName) {
				requirements.Cluster.VaultSAName = os.Getenv(config.RequirementVaultServiceAccountName)
			}
			if "" != os.Getenv(config.RequirementKanikoServiceAccountName) {
				requirements.Cluster.KanikoSAName = os.Getenv(config.RequirementKanikoServiceAccountName)
			}
			if "" != os.Getenv(config.RequirementDomainIssuerURL) {
				requirements.Ingress.DomainIssuerURL = os.Getenv(config.RequirementDomainIssuerURL)
			}
			if "" != os.Getenv(config.RequirementKaniko) {
				kaniko := os.Getenv(config.RequirementKaniko)
				if kaniko == "true" {
					requirements.Kaniko = true
				}
			}
		}
		msg := "please specify '%s' in jx-requirements when running  in  batch mode"
		if requirements.Cluster.Provider == "" {
			return nil, errors.Errorf(msg, "provider")
		}
		if requirements.Cluster.ProjectID == "" {
			return nil, errors.Errorf(msg, "project")
		}
		if requirements.Cluster.Zone == "" {
			return nil, errors.Errorf(msg, "zone")
		}
		if requirements.Cluster.EnvironmentGitOwner == "" {
			return nil, errors.Errorf(msg, "environmentGitOwner")
		}
		if requirements.Cluster.ClusterName == "" {
			return nil, errors.Errorf(msg, "clusterName")
		}
	}
	var err error
	if requirements.Cluster.Provider == "" {
		requirements.Cluster.Provider, err = util.PickName(cloud.KubernetesProviders, "Select Kubernetes provider", "the type of Kubernetes installation", o.In, o.Out, o.Err)
		if err != nil {
			return nil, errors.Wrap(err, "selecting Kubernetes provider")
		}
	}

	if requirements.Cluster.Provider == cloud.GKE {
		var currentProject, currentZone, currentClusterName string
		autoAcceptDefaults := false
		if requirements.Cluster.ProjectID == "" || requirements.Cluster.Zone == "" || requirements.Cluster.ClusterName == "" {
			kubeConfig, _, err := o.Kube().LoadConfig()
			if err != nil {
				return nil, errors.Wrapf(err, "loading kubeconfig")
			}
			context := kube.Cluster(kubeConfig)
			currentProject, currentZone, currentClusterName, err = gke.ParseContext(context)
			if err != nil {
				return nil, errors.Wrapf(err, "")
			}
			if currentClusterName != "" && currentProject != "" && currentZone != "" {
				log.Logger().Infof("")
				log.Logger().Infof("Currently connected cluster is %s in %s in project %s", util.ColorInfo(currentClusterName), util.ColorInfo(currentZone), util.ColorInfo(currentProject))
				autoAcceptDefaults = util.Confirm(fmt.Sprintf("Do you want to jx boot the %s cluster?", util.ColorInfo(currentClusterName)), true, "Enter Y to use the currently connected cluster or enter N to specify a different cluster", o.In, o.Out, o.Err)
			} else {
				log.Logger().Infof("Enter the cluster you want to jx boot")
			}
		}

		if requirements.Cluster.ProjectID == "" {
			if autoAcceptDefaults && currentProject != "" {
				requirements.Cluster.ProjectID = currentProject
			} else {
				requirements.Cluster.ProjectID, err = o.GetGoogleProjectID(currentProject)
				if err != nil {
					return nil, errors.Wrap(err, "getting project ID")
				}
			}
		}
		if requirements.Cluster.Zone == "" {
			if autoAcceptDefaults && currentZone != "" {
				requirements.Cluster.Zone = currentZone
			} else {
				requirements.Cluster.Zone, err = o.GetGoogleZone(requirements.Cluster.ProjectID, currentZone)
				if err != nil {
					return nil, errors.Wrap(err, "getting GKE Zone")
				}
			}
		}
		if requirements.Cluster.ClusterName == "" {
			if autoAcceptDefaults && currentClusterName != "" {
				requirements.Cluster.ClusterName = currentClusterName
			} else {
				requirements.Cluster.ClusterName, err = util.PickValue("Cluster name", currentClusterName, true,
					"The name for your cluster", o.In, o.Out, o.Err)
				if err != nil {
					return nil, errors.Wrap(err, "getting cluster name")
				}
			}

		}
		if !autoAcceptDefaults {
			if !o.WorkloadIdentity && !o.InCluster() {
				// connect to the specified cluster if different from the currently connected one
				log.Logger().Infof("Connecting to cluster %s", util.ColorInfo(requirements.Cluster.ClusterName))
				err = o.GCloud().ConnectToCluster(requirements.Cluster.ProjectID, requirements.Cluster.Zone, requirements.Cluster.ClusterName)
				if err != nil {
					return nil, err
				}
			} else {
				log.Logger().Info("no need to reconnect to cluster")
			}
		}
	} else {
		// lets check we want to try installation as we've only tested on GKE at the moment
		confirmed := util.Confirm("jx boot has only be validated on GKE, we'd love feedback and contributions for other Kubernetes providers",
			true, "", o.In, o.Out, o.Err)
		if !confirmed {
			return nil, nil
		}
	}

	if requirements.Cluster.EnvironmentGitOwner == "" {
		requirements.Cluster.EnvironmentGitOwner, err = util.PickValue(
			"Git Owner name for environment repositories",
			"",
			true,
			"Jenkins X leverages GitOps to track and control what gets deployed into environments.  This "+
				"requires a Git repository per environment.  This question is asking for the Git Owner where these"+
				"repositories will live",
			o.In, o.Out, o.Err)
		if err != nil {
			return nil, errors.Wrap(err, "getting GKE Zone")
		}
	}

	requirements.Cluster.Provider = strings.TrimSpace(strings.ToLower(requirements.Cluster.Provider))
	requirements.Cluster.ProjectID = strings.TrimSpace(requirements.Cluster.ProjectID)
	requirements.Cluster.Zone = strings.TrimSpace(strings.ToLower(requirements.Cluster.Zone))
	requirements.Cluster.ClusterName = strings.TrimSpace(strings.ToLower(requirements.Cluster.ClusterName))
	requirements.Cluster.EnvironmentGitOwner = strings.TrimSpace(strings.ToLower(requirements.Cluster.EnvironmentGitOwner))

	// lets fix up any missing or incorrect git kinds for public git servers
	if gits.IsGitHubServerURL(requirements.Cluster.GitServer) {
		requirements.Cluster.GitKind = "github"
	} else if gits.IsGitLabServerURL(requirements.Cluster.GitServer) {
		requirements.Cluster.GitKind = "gitlab"
	}

	requirements.SaveConfig(requirementsFileName)

	if requirements.Cluster.EnvironmentGitPrivate {
		log.Logger().Infof("Will create %s environment repos, if you want to create %s environment repos, please set %s to %s in jx-requirements.yaml", util.ColorInfo("private"), util.ColorInfo("public"), util.ColorInfo("environmentGitPrivate"), util.ColorInfo("false"))
	} else {
		log.Logger().Infof("Will create %s environment repos, if you want to create %s environment repos, please set %s to %s jx-requirements.yaml", util.ColorInfo("public"), util.ColorInfo("private"), util.ColorInfo("environmentGitPrivate"), util.ColorInfo("true"))
	}

	return requirements, nil
}

// verifyStorage verifies the associated buckets exist or if enabled lazily create them
func (o *StepVerifyPreInstallOptions) verifyStorage(requirements *config.RequirementsConfig, requirementsFileName string) error {
	log.Logger().Debug("Verifying Storage...")
	storage := &requirements.Storage
	err := o.verifyStorageEntry(requirements, requirementsFileName, &storage.Logs, "logs", "Long term log storage")
	if err != nil {
		return err
	}
	err = o.verifyStorageEntry(requirements, requirementsFileName, &storage.Reports, "reports", "Long term report storage")
	if err != nil {
		return err
	}
	err = o.verifyStorageEntry(requirements, requirementsFileName, &storage.Repository, "repository", "Chart repository")
	if err != nil {
		return err
	}
	log.Logger().Infof("the storage looks good")
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyTLS(requirements *config.RequirementsConfig) error {
	if !requirements.Ingress.TLS.Enabled {
		profile := config.LoadActiveInstallProfile()
		// silently ignore errors as they most likely because team settings aren't available
		teamSettings, err := o.TeamSettings()
		if err == nil {
			// then team settings are available
			if teamSettings.Profile != "" {
				profile = teamSettings.Profile
			}
		}

		url := "https://jenkins-x.io/architecture/tls"
		if profile == config.CloudBeesProfile {
			url = "https://go.cloudbees.com/docs/cloudbees-jenkins-x-distribution/tls/"
		}
		confirm := false
		if requirements.SecretStorage == config.SecretStorageTypeVault {
			log.Logger().Warnf("Vault is enabled and TLS is not enabled. This means your secrets will be sent to and from your cluster in the clear. See %s for more information", url)
			confirm = true
		}
		if requirements.Webhook != config.WebhookTypeNone {
			log.Logger().Warnf("TLS is not enabled so your webhooks will be called using HTTP. This means your webhook secret will be sent to your cluster in the clear. See %s for more information", url)
			confirm = true
		}
		if os.Getenv(boot.OverrideTLSWarningEnvVarName) == "true" {
			confirm = false
		}
		if confirm && !o.BatchMode {

			message := fmt.Sprintf("Do you wish to continue?")
			help := fmt.Sprintf("Jenkins X needs TLS enabled to send secrets securely. We strongly recommend enabling TLS.")
			value := util.Confirm(message, false, help, o.In, o.Out, o.Err)
			if !value {
				return errors.Errorf("cannot continue because TLS is not enabled.")
			}
		}

	}
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyStorageEntry(requirements *config.RequirementsConfig, requirementsFileName string, storageEntryConfig *config.StorageEntryConfig, name string, text string) error {
	kubeProvider := requirements.Cluster.Provider
	if !storageEntryConfig.Enabled {
		if requirements.IsCloudProvider() {
			log.Logger().Warnf("Your requirements have not enabled cloud storage for %s - we recommend enabling this for kubernetes provider %s", name, kubeProvider)
		}
		return nil
	}

	provider := factory.NewBucketProvider(requirements)

	if storageEntryConfig.URL == "" {
		// lets allow the storage bucket to be entered or created
		if o.BatchMode {
			log.Logger().Warnf("No URL provided for storage: %s", name)
			return nil
		}
		scheme := buckets.KubeProviderToBucketScheme(kubeProvider)
		if scheme == "" {
			scheme = "s3"
		}
		message := fmt.Sprintf("%s bucket URL. Press enter to create and use a new bucket", text)
		help := fmt.Sprintf("please enter the URL of the bucket to use for storage using the format %s://<bucket-name>", scheme)
		value, err := util.PickValue(message, "", false, help, o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrapf(err, "failed to pick storage bucket for %s", name)
		}

		if value == "" {
			if provider == nil {
				log.Logger().Warnf("the kubernetes provider %s has no BucketProvider in jx yet so we cannot lazily create buckets", kubeProvider)
				log.Logger().Warnf("long term storage for %s will be disabled until you provide an existing bucket URL", name)
				return nil
			}
			safeClusterName := naming.ToValidName(requirements.Cluster.ClusterName)
			safeName := naming.ToValidName(name)
			value, err = provider.CreateNewBucketForCluster(safeClusterName, safeName)
			if err != nil {
				return errors.Wrapf(err, "failed to create a dynamic bucket for cluster %s and name %s", safeClusterName, safeName)
			}
		}
		if value != "" {
			storageEntryConfig.URL = value

			err = requirements.SaveConfig(requirementsFileName)
			if err != nil {
				return errors.Wrapf(err, "failed to save changes to file: %s", requirementsFileName)
			}
		}
	}

	if storageEntryConfig.URL != "" {
		if provider == nil {
			log.Logger().Warnf("the kubernetes provider %s has no BucketProvider in jx yet - so you have to manually setup and verify your bucket URLs exist", kubeProvider)
			log.Logger().Infof("please verify this bucket exists: %s", util.ColorInfo(storageEntryConfig.URL))
			return nil
		}

		err := provider.EnsureBucketIsCreated(storageEntryConfig.URL)
		if err != nil {
			return errors.Wrapf(err, "failed to ensure the bucket URL %s is created", storageEntryConfig.URL)
		}
	}
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyProwConfigMaps(kubeClient kubernetes.Interface, ns string) error {
	err := o.verifyConfigMapExists(kubeClient, ns, "config", "config.yaml", "pod_namespace: jx")
	if err != nil {
		return err
	}
	return o.verifyConfigMapExists(kubeClient, ns, "plugins", "plugins.yaml", "cat: {}")
}

func (o *StepVerifyPreInstallOptions) verifyConfigMapExists(kubeClient kubernetes.Interface, ns string, name string, key string, defaultValue string) error {
	info := util.ColorInfo
	configMapInterface := kubeClient.CoreV1().ConfigMaps(ns)
	cm, err := configMapInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		// lets try create it
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Data: map[string]string{
				key: defaultValue,
			},
		}
		cm, err = configMapInterface.Create(cm)
		if err != nil {
			// maybe someone else just created it - lets try one more time
			cm2, err2 := configMapInterface.Get(name, metav1.GetOptions{})
			if err == nil {
				log.Logger().Infof("created ConfigMap %s in namespace %s", info(name), info(ns))
			}
			if err2 != nil {
				return fmt.Errorf("failed to create the ConfigMap %s in namespace %s due to: %s - we cannot get it either: %s", name, ns, err.Error(), err2.Error())
			}
			cm = cm2
			err = nil
		}
	}
	if err != nil {
		return err
	}

	// lets verify that there is an entry
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	_, ok := cm.Data[key]
	if !ok {
		cm.Data[key] = defaultValue
		cm.Name = name

		_, err = configMapInterface.Update(cm)
		if err != nil {
			return fmt.Errorf("failed to update the ConfigMap %s in namespace %s to add key %s due to: %s", name, ns, key, err.Error())
		}
	}
	log.Logger().Infof("verified there is a ConfigMap %s in namespace %s", info(name), info(ns))
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyIngress(requirements *config.RequirementsConfig, requirementsFileName string) error {
	log.Logger().Debug("Verifying Ingress...")
	domain := requirements.Ingress.Domain
	if requirements.Ingress.IsAutoDNSDomain() {
		log.Logger().Infof("clearing the domain %s as when using auto-DNS domains we need to regenerate to ensure its always accurate in case the cluster or ingress service is recreated", util.ColorInfo(domain))
		requirements.Ingress.Domain = ""
		err := requirements.SaveConfig(requirementsFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to save changes to file: %s", requirementsFileName)
		}
	}
	return nil
}

func modifyMapIfNotBlank(m map[string]string, key string, value string) {
	if m != nil {
		if value != "" {
			m[key] = value
		} else {
			log.Logger().Debugf("Cannot update key %s, value is nil", key)
		}
	} else {
		log.Logger().Debugf("Cannot update key %s, map is nil", key)
	}
}
