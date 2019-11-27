package boot

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
)

// StepBootVaultOptions contains the command line flags
type StepBootVaultOptions struct {
	*opts.CommonOptions
	Dir               string
	ProviderValuesDir string
	Namespace         string
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
		Use:     "vault",
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
	cmd.Flags().StringVarP(&o.ProviderValuesDir, "provider-values-dir", "", "", "The optional directory of kubernetes provider specific files")
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

	kubeClient, err := o.KubeClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create kubernetes client")
	}

	if requirements.Vault.Name == "" {
		requirements.Vault.Name = kubevault.SystemVaultNameForCluster(requirements.Cluster.ClusterName)
	}

	log.Logger().Debugf("Using vault name '%s'", requirements.Vault.Name)

	noExposeVault, err := o.verifyVaultIngress(requirements, kubeClient, ns, requirements.Vault.Name)
	if err != nil {
		return err
	}

	ic, err := o.createIngressConfig(requirements)
	if err != nil {
		return err
	}

	// verify configuration
	copyOptions := *o.CommonOptions
	copyOptions.BatchMode = true

	// setting this to true so that we disable the use of exposecontroller with vault
	// TODO: do we need to wait for certificates to be available via the secret
	noExposeVault = true

	cvo := &create.CreateVaultOptions{
		CreateOptions: create.CreateOptions{
			CommonOptions: &copyOptions,
		},

		Namespace:           ns,
		RecreateVaultBucket: requirements.Vault.RecreateBucket,
		IngressConfig:       ic,
		NoExposeVault:       noExposeVault,
		BucketName:          requirements.Vault.Bucket,
		KeyringName:         requirements.Vault.Keyring,
		KeyName:             requirements.Vault.Key,
		ServiceAccountName:  requirements.Vault.ServiceAccount,
		ClusterName:         requirements.Cluster.ClusterName,
	}
	if requirements.Cluster.Provider == cloud.EKS {
		if requirements.Vault.AWSConfig != nil {
			awsConfig := requirements.Vault.AWSConfig
			cvo.AWSCreateVaultOptions = create.AWSCreateVaultOptions{}
			secretAccessKey := os.Getenv("VAULT_AWS_SECRET_ACCESS_KEY")
			accessKeyId := os.Getenv("VAULT_AWS_ACCESS_KEY_ID")
			if !awsConfig.AutoCreate && (checkRequiredResource("dynamoDBTable", awsConfig.DynamoDBTable) ||
				checkRequiredResource("secretAccessKey", secretAccessKey) ||
				checkRequiredResource("accessKeyId", accessKeyId) ||
				checkRequiredResource("kmsKeyId", awsConfig.KMSKeyID) ||
				checkRequiredResource("s3Bucket", awsConfig.S3Bucket)) {
				log.Logger().Info("Some of the required provided values are empty - We will create all resources")
				awsConfig.AutoCreate = true
			}
			cvo.Boot = true
			cvo.AWSConfig = kubevault.AWSConfig{
				AWSUnsealConfig: v1alpha1.AWSUnsealConfig{
					KMSKeyID:  awsConfig.KMSKeyID,
					KMSRegion: awsConfig.KMSRegion,
					S3Bucket:  awsConfig.S3Bucket,
					S3Prefix:  awsConfig.S3Prefix,
					S3Region:  awsConfig.S3Region,
				},
				AutoCreate:          awsConfig.AutoCreate,
				DynamoDBTable:       awsConfig.DynamoDBTable,
				DynamoDBRegion:      awsConfig.DynamoDBRegion,
				AccessKeyID:         accessKeyId,
				SecretAccessKey:     secretAccessKey,
				ProvidedIAMUsername: awsConfig.ProvidedIAMUsername,
			}
		}

		cvo.AWSTemplatesDir = filepath.Join(o.Dir, o.ProviderValuesDir, cloud.EKS, "templates")

	}
	// first argument is the vault name
	cvo.Args = []string{requirements.Vault.Name}
	cvo.SetDevNamespace(ns)

	provider := requirements.Cluster.Provider
	if provider == cloud.GKE {
		if cvo.GKEProjectID == "" {
			cvo.GKEProjectID = requirements.Cluster.ProjectID
		}
		if cvo.GKEProjectID == "" {
			return config.MissingRequirement("project", requirementsFile)
		}

		if cvo.GKEZone == "" {
			cvo.GKEZone = requirements.Cluster.Zone
		}
		if cvo.GKEZone == "" {
			return config.MissingRequirement("zone", requirementsFile)
		}
	} else if provider == cloud.AWS || provider == cloud.EKS {
		defaultRegion := requirements.Cluster.Region
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

	tag, err := o.vaultOperatorImageTag(&requirements.VersionStream)
	if err != nil {
		return err
	}
	opts := o.CommonOptions
	values := []string{
		"image.repository=" + kubevault.VaultOperatorImage,
		"image.tag=" + tag,
	}
	opts.SetValues = strings.Join(values, ",")

	log.Logger().Infof("installing vault operator with helm values:  %s", util.ColorInfo(opts.SetValues))
	err = create.InstallVaultOperator(opts, ns, &requirements.VersionStream)
	if err != nil {
		return errors.Wrap(err, "unable to install vault operator")
	}

	log.Logger().Infof("vault operator installed in namespace %s", ns)

	// Create a new System vault
	vaultOperatorClient, err := cvo.VaultOperatorClient()
	if err != nil {
		return err
	}

	// lets store the system vault name
	_, err = kube.DefaultModifyConfigMap(kubeClient, ns, kube.ConfigMapNameJXInstallConfig,
		func(configMap *corev1.ConfigMap) error {
			configMap.Data[kube.SystemVaultName] = requirements.Vault.Name
			configMap.Data[secrets.SecretsLocationKey] = string(secrets.VaultLocationKind)
			return nil
		}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving secrets location in ConfigMap %s in namespace %s", kube.ConfigMapNameJXInstallConfig, ns)
	}

	log.Logger().Infof("finding vault '%s' in namespace %s", requirements.Vault.Name, ns)

	if kubevault.FindVault(vaultOperatorClient, requirements.Vault.Name, ns) {
		log.Logger().Infof("System vault named %s in namespace %s already exists",
			util.ColorInfo(requirements.Vault.Name), util.ColorInfo(ns))
	} else {
		log.Logger().Info("Creating new system vault")
		err = cvo.CreateVault(vaultOperatorClient, requirements.Vault.Name, provider)
		if err != nil {
			return err
		}
		log.Logger().Infof("System vault created named %s in namespace %s.",
			util.ColorInfo(requirements.Vault.Name), util.ColorInfo(ns))
	}
	return nil
}

func (o *StepBootVaultOptions) createIngressConfig(requirements *config.RequirementsConfig) (kube.IngressConfig, error) {
	i := requirements.Ingress
	tls := i.TLS
	ic := kube.IngressConfig{
		Domain:  i.Domain,
		Exposer: "Ingress",
		Email:   tls.Email,
		TLS:     tls.Enabled,
	}
	return ic, nil
}

// verifyVaultIngress verifies there is a Vault ingress and if not create one if there is a file at
func (o *StepBootVaultOptions) verifyVaultIngress(requirements *config.RequirementsConfig, kubeClient kubernetes.Interface, ns string, systemVaultName string) (bool, error) {
	fileName := filepath.Join(o.Dir, "vault-ing.tmpl.yaml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	if !exists {
		log.Logger().Warnf("failed to find file %s\n", fileName)
		return false, nil
	}
	data, err := readYamlTemplate(fileName, requirements)
	if err != nil {
		return true, errors.Wrapf(err, "failed to load vault ingress template file %s", fileName)
	}

	log.Logger().Infof("applying vault ingress in namespace %s for vault name %s\n", util.ColorInfo(ns), util.ColorInfo(systemVaultName))

	tmpFile, err := ioutil.TempFile("", "vault-ingress-")
	if err != nil {
		return true, errors.Wrapf(err, "failed to create temporary file for vault YAML")
	}

	tmpFileName := tmpFile.Name()
	err = ioutil.WriteFile(tmpFileName, data, util.DefaultWritePermissions)
	if err != nil {
		return true, errors.Wrapf(err, "failed to save vault ingress YAML file %s", tmpFileName)
	}

	args := []string{"apply", "--force", "-f", tmpFileName, "-n", ns}
	err = o.RunCommand("kubectl", args...)
	if err != nil {
		return true, errors.Wrapf(err, "failed to apply vault ingress YAML")
	}
	return true, nil
}

// vaultOperatorImageTag lookups the vault operator image tag in the version stream
func (o *StepBootVaultOptions) vaultOperatorImageTag(versionStream *config.VersionStreamConfig) (string, error) {
	resolver, err := o.CreateVersionResolver(versionStream.URL, versionStream.Ref)
	if err != nil {
		return "", errors.Wrap(err, "creating the vault-operator docker image version resolver")
	}
	fullImage, err := resolver.ResolveDockerImage(kubevault.VaultOperatorImage)
	if err != nil {
		return "", errors.Wrapf(err, "looking up the vault-operator %q image into the version stream",
			kubevault.VaultOperatorImage)
	}
	parts := strings.Split(fullImage, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("no tag found for image %q in version stream", kubevault.VaultOperatorImage)
	}
	return parts[1], nil
}

// readYamlTemplate evaluates the given go template file and returns the output data
func readYamlTemplate(templateFile string, requirements *config.RequirementsConfig) ([]byte, error) {
	_, name := filepath.Split(templateFile)
	funcMap := helm.NewFunctionMap()
	tmpl, err := template.New(name).Option("missingkey=error").Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Ingress template: %s", templateFile)
	}

	requirementsMap, err := requirements.ToMap()
	if err != nil {
		return nil, errors.Wrapf(err, "failed turn requirements into a map: %v", requirements)
	}

	templateData := map[string]interface{}{
		"Requirements": chartutil.Values(requirementsMap),
		"Environments": chartutil.Values(requirements.EnvironmentMap()),
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute Ingress template: %s", templateFile)
	}
	data := buf.Bytes()
	return data, nil
}

func checkRequiredResource(resourceName string, value string) bool {
	if value == "" {
		log.Logger().Warnf("Vault.AutoCreate is false but required property %s is missing", resourceName)
		return true
	}
	return false
}
