package boot

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// StepBootVaultOptions contains the command line flags
type StepBootVaultOptions struct {
	*opts.CommonOptions
	Dir       string
	Namespace string
}

var (
	stepBootVaultLong = templates.LongDesc(`
		This step boots up Vault in the current cluster if its enabled in the 'jx-requirements.yml' file and is not already installed.

		This step is intended to be used in the Jenkins X Boot Pipeline: https://jenkins-x.io/getting-started/boot/
`)

	stepBootVaultExample = templates.Examples(`
		# boots up Vault if its required
		jx step boot vault
`)
)

// NewCmdStepBootVault creates the command
func NewCmdStepBootVault(commonOpts *opts.CommonOptions) *cobra.Command {
	o := StepBootVaultOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "boot vault",
		Short:   "This step boots up Vault in the current cluster if its enabled in the 'jx-requirements.yml' file and is not already installed",
		Long:    stepBootVaultLong,
		Example: stepBootVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", fmt.Sprintf("the directory to look for the requirements file: %s", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace that Jenkins X will be booted into. If not specified it defaults to $DEPLOY_NAMESPACE")

	return cmd
}

// Run runs the command
func (o *StepBootVaultOptions) Run() error {
	ns, err := o.GetDeployNamespace(o.Namespace)
	if err != nil {
		return err
	}
	o.SetDevNamespace(ns)

	requirements, requirementsFile, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}

	info := util.ColorInfo
	if requirements.SecretStorage != config.SecretStorageTypeVault {
		log.Logger().Infof("not attempting to boot Vault as using secret storage: %s\n", info(string(requirements.SecretStorage)))
		return nil
	}

	ic, err := o.loadIngressConfig()
	if err != nil {
		return err
	}

	// verify configuration
	copyOptions := *o.CommonOptions
	copyOptions.BatchMode = true

	cvo := &create.CreateVaultOptions{
		CreateOptions: create.CreateOptions{
			CommonOptions: &copyOptions,
		},
		Namespace:           ns,
		RecreateVaultBucket: true,
		IngressConfig:       ic,
		// TODO - load from a local yaml file if available?
		// AWSConfig:           o.AWSConfig,
	}
	cvo.SetDevNamespace(ns)

	provider := requirements.Provider
	if provider == cloud.GKE {
		if cvo.GKEProjectID == "" {
			cvo.GKEProjectID = requirements.ProjectID
		}
		if cvo.GKEProjectID == "" {
			return config.MissingRequirement("project", requirementsFile)
		}

		if cvo.GKEZone == "" {
			cvo.GKEZone = requirements.Zone
		}
		if cvo.GKEZone == "" {
			return config.MissingRequirement("zone", requirementsFile)
		}
	} else if provider == cloud.AWS || provider == cloud.EKS {
		defaultRegion := requirements.Region
		if cvo.DynamoDBRegion == "" {
			cvo.DynamoDBRegion = defaultRegion
			log.Logger().Infof("Region not specified for DynamoDB, defaulting to %s", util.ColorInfo(defaultRegion))
		}
		if cvo.KMSRegion == "" {
			cvo.KMSRegion = defaultRegion
			log.Logger().Infof("Region not specified for KMS, defaulting to %s", util.ColorInfo(defaultRegion))

		}
		if cvo.S3Region == "" {
			cvo.S3Region = defaultRegion
			log.Logger().Infof("Region not specified for S3, defaulting to %s", util.ColorInfo(defaultRegion))
		}
		if defaultRegion == "" {
			return config.MissingRequirement("region", requirementsFile)
		}
	}

	err = create.InstallVaultOperator(o.CommonOptions, ns)
	if err != nil {
		return errors.Wrap(err, "unable to install vault operator")
	}

	// Create a new System vault
	vaultOperatorClient, err := cvo.VaultOperatorClient()
	if err != nil {
		return err
	}

	systemVaultName, err := kubevault.SystemVaultName(o.Kube())
	if err != nil {
		return errors.Wrap(err, "building the system vault name from cluster name")
	}

	kubeClient, err := o.KubeClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client")
	}

	// lets store the system vault name
	_, err = kube.DefaultModifyConfigMap(kubeClient, ns, kube.ConfigMapNameJXInstallConfig,
		func(configMap *corev1.ConfigMap) error {
			configMap.Data[kube.SystemVaultName] = systemVaultName
			configMap.Data[secrets.SecretsLocationKey] = string(secrets.VaultLocationKind)
			return nil
		}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving secrets location in ConfigMap %s in namespace %s", kube.ConfigMapNameJXInstallConfig, ns)
	}

	if kubevault.FindVault(vaultOperatorClient, systemVaultName, ns) {
		log.Logger().Infof("System vault named %s in namespace %s already exists",
			util.ColorInfo(systemVaultName), util.ColorInfo(ns))
	} else {
		log.Logger().Info("Creating new system vault")
		err = cvo.CreateVault(vaultOperatorClient, systemVaultName, provider)
		if err != nil {
			return err
		}
		log.Logger().Infof("System vault created named %s in namespace %s.",
			util.ColorInfo(systemVaultName), util.ColorInfo(ns))
	}
	return nil
}

func (o *StepBootVaultOptions) loadIngressConfig() (kube.IngressConfig, error) {
	ic := kube.IngressConfig{}

	// lets try load the generated ingress `env/cluster/values.yaml` file
	fileName := filepath.Join(o.Dir, "..", "..", "env", "cluster", "values.yaml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return ic, errors.Wrapf(err, "failed to check for file %s", fileName)
	}
	if exists {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return ic, errors.Wrapf(err, "failed to load file %s", fileName)
		}
		err = yaml.Unmarshal(data, &ic)
		if err != nil {
			return ic, errors.Wrapf(err, "failed to unmarshal YAML file %s due to %s\n", fileName, err.Error())
		}
	} else {
		log.Logger().Warnf("No ingress cluster configuration file exists at %s\n", fileName)
	}

	if ic.Exposer == "" {
		ic.Exposer = "Ingress"
	}
	log.Logger().Infof("loaded ingres config: %#v\n", ic)
	return ic, nil

}
