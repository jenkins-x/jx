package verify

import (
	"fmt"
	"strings"

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
	flag := strings.ToLower(o.LazyCreateFlag)
	if flag != "" {
		if flag == "true" {
			o.LazyCreate = true
		} else if flag == "false" {
			o.LazyCreate = false
		} else {
			return util.InvalidOption("lazy-create", flag, []string{"true", "false"})
		}
	} else {
		// lets default from the requirements
		if !requirements.Terraform {
			o.LazyCreate = true
		}
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

	err = o.verifyInstallConfig(kubeClient, ns, requirements)
	if err != nil {
		return err
	}

	err = o.verifyStorage(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	if requirements.Kaniko {
		if requirements.Provider == cloud.GKE {
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
	io.Flags.Provider = requirements.Provider
	io.SetInstallValues(map[string]string{
		kube.ClusterName: requirements.ClusterName,
		kube.ProjectID:   requirements.ProjectID,
		kube.Domain:      requirements.Ingress.Domain,
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
func (o *StepVerifyPreInstallOptions) verifyInstallConfig(kubeClient kubernetes.Interface, ns string, requirements *config.RequirementsConfig) error {
	_, err := kube.DefaultModifyConfigMap(kubeClient, ns, kube.ConfigMapNameJXInstallConfig,
		func(configMap *corev1.ConfigMap) error {
			secretsLocation := string(secrets.FileSystemLocationKind)
			if requirements.SecretStorage == config.SecretStorageTypeVault {
				secretsLocation = string(secrets.VaultLocationKind)
			}

			modifyMapIfNotBlank(configMap.Data, kube.KubeProvider, requirements.Provider)
			modifyMapIfNotBlank(configMap.Data, kube.ProjectID, requirements.ProjectID)
			modifyMapIfNotBlank(configMap.Data, kube.ClusterName, requirements.ClusterName)
			modifyMapIfNotBlank(configMap.Data, kube.Domain, requirements.Ingress.Domain)
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
	err := o.verifyStorageEntry(requirements, requirementsFileName, &storage.Logs, "Logs")
	if err != nil {
		return err
	}
	err = o.verifyStorageEntry(requirements, requirementsFileName, &storage.Reports, "Reports")
	if err != nil {
		return err
	}
	err = o.verifyStorageEntry(requirements, requirementsFileName, &storage.Repository, "Repository")
	if err != nil {
		return err
	}
	log.Logger().Infof("the storage looks good\n")
	return nil
}

func (o *StepVerifyPreInstallOptions) verifyStorageEntry(requirements *config.RequirementsConfig, requirementsFileName string, storageEntryConfig *config.StorageEntryConfig, name string) error {
	kubeProvider := requirements.Provider
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
		message := fmt.Sprintf("storage bucket URL (%s://bucketName) for %s: ", scheme, name)
		help := fmt.Sprintf("please enter the URL of the bucket to use for storage like %s://bucketName or leave blank to disable cloud storage", scheme)
		value, err := util.PickValue(message, "", false, help, o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrapf(err, "failed to pick storage bucket for %s", name)
		}

		if value == "" {
			if provider == nil {
				log.Logger().Warnf("the kubernete provider %s has no BucketProvider in jx yet so we cannot lazily create buckets - so long term stor\n", kubeProvider)
				log.Logger().Warnf("long term storage for %s will be disabled until you provide an existing bucket URL\n", name)
				return nil
			}
			safeClusterName := naming.ToValidName(requirements.ClusterName)
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
			log.Logger().Warnf("the kubernete provider %s has no BucketProvider in jx yet - so you have to manually setup and verify your bucket URLs exist\n", kubeProvider)
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

func modifyMapIfNotBlank(m map[string]string, key string, value string) {
	if value != "" {
		m[key] = value
	}
}
