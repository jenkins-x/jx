package verify

import (
	"fmt"
	"strings"

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
	Debug          bool
	Dir            string
	LazyCreate     bool
	LazyCreateFlag string
	Namespace      string
}

// NewCmdStepVerifyPreInstall creates the `jx step verify pod` command
func NewCmdStepVerifyPreInstall(commonOpts *opts.CommonOptions) *cobra.Command {

	options := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: opts.StepOptions{
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
	return cmd
}

// Run implements this command
func (o *StepVerifyPreInstallOptions) Run() error {
	info := util.ColorInfo
	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
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

	o.SetDevNamespace(ns)

	log.Logger().Infof("verifying the kubernetes cluster before we try to boot Jenkins X in namespace: %s\n", info(ns))
	if o.LazyCreate {
		log.Logger().Infof("we will try to lazily create any missing resources to get the current cluster ready to boot Jenkins X\n")
	} else {
		log.Logger().Infof("lazy create of cloud resources is disabled\n")

	}

	err = o.verifyDevNamespace(kubeClient, ns)
	if err != nil {
		if o.LazyCreate {
			log.Logger().Infof("attempting to lazily create the deploy namespace %s\n", info(ns))

			err = kube.EnsureDevNamespaceCreatedWithoutEnvironment(kubeClient, ns)
			if err != nil {
				return errors.Wrapf(err, "failed to lazily create the namespace %s", ns)
			}
			// lets rerun the verify step to ensure its all sorted now
			err = o.verifyDevNamespace(kubeClient, ns)
		}
	}
	if err != nil {
		return err
	}

	no := &namespace.NamespaceOptions{}
	no.CommonOptions = o.CommonOptions
	no.Args = []string{ns}
	log.Logger().Infof("setting the local kubernetes context to the deploy namespace %s\n", info(ns))
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

	err = o.verifyInstallConfig(kubeClient, ns, requirements, requirementsFileName)
	if err != nil {
		return err
	}

	err = o.verifyStorage(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	if requirements.Kaniko {
		if requirements.Cluster.Provider == cloud.GKE {
			log.Logger().Infof("validating the kaniko secret in namespace %s\n", info(ns))

			err = o.validateKaniko(ns)
			if err != nil {
				if o.LazyCreate {
					log.Logger().Infof("attempting to lazily create the deploy namespace %s\n", info(ns))

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

	log.Logger().Infof("the cluster looks good, you are ready to '%s' now!\n", info("jx boot"))
	fmt.Println()
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyDevNamespace(kubeClient kubernetes.Interface, ns string) error {
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
	log.Logger().Infof("lazily creating the kaniko secret\n")
	io := &create.InstallOptions{}
	io.CommonOptions = o.CommonOptions
	io.Flags.Kaniko = true
	io.Flags.Namespace = ns
	io.Flags.Provider = requirements.Cluster.Provider
	io.SetInstallValues(map[string]string{
		kube.ClusterName: requirements.Cluster.ClusterName,
		kube.ProjectID:   requirements.Cluster.ProjectID,
	})
	err := io.ConfigureKaniko()
	if err != nil {
		return err
	}
	data := io.AdminSecretsService.Flags.KanikoSecret
	if data == "" {
		return fmt.Errorf("failed to create the kaniko secret data")
	}
	return o.createKanikoSecret(ns, data)
}

// verifyInstallConfig lets ensure we modify the install ConfigMap with the requirements
func (o *StepVerifyPreInstallOptions) verifyInstallConfig(kubeClient kubernetes.Interface, ns string, requirements *config.RequirementsConfig, requirementsFileName string) error {
	_, err := kube.DefaultModifyConfigMap(kubeClient, ns, kube.ConfigMapNameJXInstallConfig,
		func(configMap *corev1.ConfigMap) error {
			secretsLocation := string(secrets.FileSystemLocationKind)
			if requirements.SecretStorage == config.SecretStorageTypeVault {
				secretsLocation = string(secrets.VaultLocationKind)
			}

			if o.BatchMode {
				msg := "please specify '%s' in jx-requirements when running  in  batch mode"
				if requirements.Cluster.Provider == "" {
					return errors.Errorf(msg, "provider")
				}
				if requirements.Cluster.ProjectID == "" {
					return errors.Errorf(msg, "project")
				}
				if requirements.Cluster.Zone == "" {
					return errors.Errorf(msg, "zone")
				}
				if requirements.Cluster.EnvironmentGitOwner == "" {
					return errors.Errorf(msg, "environmentGitOwner")
				}
				if requirements.Cluster.ClusterName == "" {
					return errors.Errorf(msg, "clusterName")
				}
			}
			var err error
			if requirements.Cluster.Provider == "" {
				requirements.Cluster.Provider, err = util.PickName(cloud.KubernetesProviders, "Select Kubernetes provider", "the type of Kubernetes installation", o.In, o.Out, o.Err)
				if err != nil {
					return errors.Wrap(err, "selecting Kubernetes provider")
				}
			}

			if requirements.Cluster.Provider == cloud.GKE {
				var currentProject, currentZone, currentClusterName string
				autoAcceptDefaults := false
				if requirements.Cluster.ProjectID == "" || requirements.Cluster.Zone == "" || requirements.Cluster.ClusterName == "" {
					kubeConfig, _, err := o.Kube().LoadConfig()
					if err != nil {
						return errors.Wrapf(err, "loading kubeconfig")
					}
					context := kube.Cluster(kubeConfig)
					currentProject, currentZone, currentClusterName, err = gke.ParseContext(context)
					if err != nil {
						return errors.Wrapf(err, "")
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
					}
				}
				if requirements.Cluster.Zone == "" {
					if autoAcceptDefaults && currentZone != "" {
						requirements.Cluster.Zone = currentZone
					} else {
						requirements.Cluster.Zone, err = o.GetGoogleZone(requirements.Cluster.ProjectID, currentZone)
						if err != nil {
							return errors.Wrap(err, "getting GKE Zone")
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
							return errors.Wrap(err, "getting GKE Zone")
						}
					}

				}
			} else {
				// lets check we want to try installation as we've only tested on GKE at the moment
				confirmed := util.Confirm("jx boot has only be validated on GKE, we'd love feedback and contributions for other Kubernetes providers",
					true, "", o.In, o.Out, o.Err)
				if !confirmed {
					return nil
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
					return errors.Wrap(err, "getting GKE Zone")
				}
			}

			requirements.Cluster.Provider = strings.TrimSpace(strings.ToLower(requirements.Cluster.Provider))
			requirements.Cluster.ProjectID = strings.TrimSpace(requirements.Cluster.ProjectID)
			requirements.Cluster.Zone = strings.TrimSpace(strings.ToLower(requirements.Cluster.Zone))
			requirements.Cluster.ClusterName = strings.TrimSpace(strings.ToLower(requirements.Cluster.ClusterName))
			requirements.Cluster.EnvironmentGitOwner = strings.TrimSpace(strings.ToLower(requirements.Cluster.EnvironmentGitOwner))

			requirements.SaveConfig(requirementsFileName)
			if err != nil {
				return err
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

// verifyStorage verifies the associated buckets exist or if enabled lazily create them
func (o *StepVerifyPreInstallOptions) verifyStorage(requirements *config.RequirementsConfig, requirementsFileName string) error {
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
	log.Logger().Infof("the storage looks good\n")
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyStorageEntry(requirements *config.RequirementsConfig, requirementsFileName string, storageEntryConfig *config.StorageEntryConfig, name string, text string) error {
	kubeProvider := requirements.Cluster.Provider
	if !storageEntryConfig.Enabled {
		if requirements.IsCloudProvider() {
			log.Logger().Warnf("Your requirements have not enabled cloud storage for %s - we recommend enabling this for kubernetes provider %s\n", name, kubeProvider)
		}
		return nil
	}

	provider := factory.NewBucketProvider(requirements)

	if storageEntryConfig.URL == "" {
		// lets allow the storage bucket to be entered or created
		if o.BatchMode {
			log.Logger().Warnf("No URL provided for storage: %s\n", name)
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
			log.Logger().Warnf("the kubernetes provider %s has no BucketProvider in jx yet - so you have to manually setup and verify your bucket URLs exist\n", kubeProvider)
			log.Logger().Infof("please verify this bucket exists: %s\n", util.ColorInfo(storageEntryConfig.URL))
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
	err := o.verifyConfigMapExists(kubeClient, ns, "config", "config.yaml", "pod_namespace: jx\n")
	if err != nil {
		return err
	}
	return o.verifyConfigMapExists(kubeClient, ns, "plugins", "plugins.yaml", "cat: {}\n")
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
				log.Logger().Infof("created ConfigMap %s in namespace %s\n", info(name), info(ns))
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
	log.Logger().Infof("verified there is a ConfigMap %s in namespace %s\n", info(name), info(ns))
	return nil
}

func modifyMapIfNotBlank(m map[string]string, key string, value string) {
	if value != "" {
		m[key] = value
	}
}
