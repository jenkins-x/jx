package verify

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
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
	requirements, _, err := config.LoadRequirementsConfig(o.Dir)
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
	o.SetDevNamespace(ns)

	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

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

	err = o.verifyInstallConfig(kubeClient, ns, requirements)
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

			modifyMapIfNotBlank(configMap.Data, kube.ProjectID, requirements.ProjectID)
			modifyMapIfNotBlank(configMap.Data, kube.ClusterName, requirements.ClusterName)
			modifyMapIfNotBlank(configMap.Data, secrets.SecretsLocationKey, secretsLocation)
			return nil
		}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving secrets location in ConfigMap %s in namespace %s", kube.ConfigMapNameJXInstallConfig, ns)
	}
	return nil
}

func modifyMapIfNotBlank(m map[string]string, key string, value string) {
	if value != "" {
		m[key] = value
	}
}
