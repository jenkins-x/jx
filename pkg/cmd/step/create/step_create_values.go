package create

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/aks"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/iks"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/secreturl"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var (
	optionSecretsScheme = "secrets-scheme"
)

var (
	createValuesLong = templates.LongDesc(`
		Creates a values.yaml from a schema
`)

	createValuesExample = templates.Examples(`
		# create the values.yaml file from values.schema.json in the current directory
		jx step create values

		# create the values.yaml file from values.schema.json in the /path/to/values directory
		jx step create values -d /path/to/values

		# create the cheese.yaml file from cheese.schema.json in the current directory 
		jx step create values --name cheese
	
			`)
)

// StepCreateValuesOptions contains the command line flags
type StepCreateValuesOptions struct {
	step.StepCreateOptions

	Dir       string
	Namespace string
	Name      string
	BasePath  string

	Schema     string
	ValuesFile string

	SecretsScheme string
}

// StepCreateValuesResults stores the generated results
type StepCreateValuesResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreateValues Creates a new Command object
func NewCmdStepCreateValues(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateValuesOptions{
		StepCreateOptions: step.StepCreateOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "values",
		Short:   "Creates the values.yaml file from a schema",
		Long:    createValuesLong,
		Example: createValuesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "the directory to look for the <kind>.schema.json and write the <kind>.yaml, defaults to the current directory")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "the namespace Jenkins X is installed into. If not specified it defaults to $DEPLOY_NAMESPACE or else defaults to the current kubernetes namespace")
	cmd.Flags().StringVarP(&options.Schema, "schema", "", "", "the path to the schema file, overrides --dir and --name")
	cmd.Flags().StringVarP(&options.Name, "name", "", "values", "the kind of the file to create (and, by default, the schema name)")
	cmd.Flags().StringVarP(&options.BasePath, "secret-base-path", "", "", fmt.Sprintf("the secret path used to store secrets in vault / file system. Typically a unique name per cluster+team. If none is specified we will default it to the cluster name from the %s file in the current or a parent directory.", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&options.ValuesFile, "out", "", "", "the path to the file to create, overrides --dir and --name")
	cmd.Flags().StringVarP(&options.SecretsScheme, optionSecretsScheme, "", "", fmt.Sprintf("the scheme to store/reference any secrets in, valid options are vault and local. If none are specified we will default it from the %s file in the current or a parent directory.", config.RequirementsConfigFileName))
	return cmd
}

// Run implements this command
func (o *StepCreateValuesOptions) Run() error {
	ns, err := o.GetDeployNamespace(o.Namespace)
	if err != nil {
		return err
	}
	o.SetDevNamespace(ns)

	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	// lets default to the install requirements setting
	requirements, fileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err := util.FileExists(fileName)
	if err != nil {
		return err
	}
	info := util.ColorInfo
	if o.SecretsScheme == "" {
		if exists {
			o.SecretsScheme = string(requirements.SecretStorage)
			log.Logger().Infof("defaulting to secret storage scheme %s found from requirements file at %s\n", info(o.SecretsScheme), info(fileName))
		} else {
			log.Logger().Warnf("there is no requirements file at %s\n", fileName)
		}
	}
	if o.BasePath == "" {
		if exists {
			o.BasePath = requirements.Cluster.ClusterName
			log.Logger().Infof("defaulting to secret base path to the cluster name %s found from requirements file at %s\n", info(o.BasePath), info(fileName))
		} else {
			log.Logger().Warnf("there is no requirements file at %s\n", fileName)
		}

	}
	if !(o.SecretsScheme == "vault" || o.SecretsScheme == "local") {
		util.InvalidArgf(optionSecretsScheme, "Use one of vault or local")
	}
	if o.Schema == "" {
		o.Schema = filepath.Join(o.Dir, fmt.Sprintf("%s.schema.json", o.Name))
	}

	err = surveyutils.TemplateSchemaFile(o.Schema, requirements)
	if err != nil {
		return errors.Wrapf(err, "failed to generate %s from template", o.Schema)
	}

	if o.ValuesFile == "" {
		o.ValuesFile = filepath.Join(o.Dir, fmt.Sprintf("%s.yaml", o.Name))
	}

	secretURLClient, err := o.GetSecretURLClient(secrets.ToSecretsLocation(o.SecretsScheme))
	if err != nil {
		return err
	}

	err = o.verifyRegistryConfig(requirements, fileName, secretURLClient)
	if err != nil {
		return err
	}

	fmt.Println()
	err = o.CreateValuesFile(secretURLClient)
	if err != nil {
		return errors.WithStack(err)
	}
	fmt.Println()
	return nil
}

// CreateValuesFile builds the clients and settings from the team needed to run apps.ProcessValues, and then copies the answer
// to the location requested by the command
func (o *StepCreateValuesOptions) CreateValuesFile(secretURLClient secreturl.Client) error {
	schema, err := ioutil.ReadFile(o.Schema)
	if err != nil {
		return errors.Wrapf(err, "reading schema %s", o.Schema)
	}
	gitOpsURL := ""
	gitOps, devEnv := o.GetDevEnv()
	if gitOps {
		gitOpsURL = devEnv.Spec.Source.URL
	}
	teamName, _, err := o.TeamAndEnvironmentNames()
	if err != nil {
		return errors.Wrapf(err, "getting team name")
	}
	existing, err := helm.LoadValuesFile(o.ValuesFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load values file %s", o.ValuesFile)
	}

	valuesFileName, cleanup, err := apps.ProcessValues(schema, o.Name, gitOpsURL, teamName, o.BasePath, o.BatchMode, false, secretURLClient, existing, o.SecretsScheme, o.GetIOFileHandles(), o.Verbose)
	defer cleanup()
	if err != nil {
		return errors.WithStack(err)
	}
	err = util.CopyFile(valuesFileName, o.ValuesFile)
	if err != nil {
		return errors.Wrapf(err, "moving %s to %s", valuesFileName, o.ValuesFile)
	}
	return nil
}

func (o *StepCreateValuesOptions) verifyRegistryConfig(requirements *config.RequirementsConfig, requirementsFileName string, secretClient secreturl.Client) error {
	log.Logger().Debug("Verifying Registry...")
	registry := requirements.Cluster.Registry
	configJSON := ""
	if registry == "" {
		kubeConfig, _, err := o.Kube().LoadConfig()
		if err != nil {
			return errors.Wrapf(err, "failed to load kube config")
		}

		// lets try default the container registry
		switch requirements.Cluster.Provider {
		case cloud.AWS, cloud.EKS:
			registry, err = amazon.GetContainerRegistryHost()
			if err != nil {
				return err
			}
		case cloud.GKE:
			if requirements.Kaniko && requirements.Webhook != config.WebhookTypeJenkins {
				registry = "gcr.io"
			}
		case cloud.AKS:
			server := kube.CurrentServer(kubeConfig)
			azureCLI := aks.NewAzureRunner()
			resourceGroup, name, cluster, err := azureCLI.GetClusterClient(server)
			if err != nil {
				return errors.Wrap(err, "getting cluster from Azure")
			}
			registryID := ""
			azureRegistrySubscription := requirements.Cluster.AzureConfig.RegistrySubscription
			if azureRegistrySubscription == "" {
				log.Logger().Info("no azure registry subscription is specified in 'jx-requirements.yml' at cluster.azure.registrySubscription")
			}
			configJSON, registry, registryID, err = azureCLI.GetRegistry(azureRegistrySubscription, resourceGroup, name, registry)
			if err != nil {
				return errors.Wrap(err, "getting registry configuration from Azure")
			}
			azureCLI.AssignRole(cluster, registryID)
			log.Logger().Infof("Assign AKS %s a reader role for ACR %s", util.ColorInfo(server), util.ColorInfo(registry))
		case cloud.IKS:
			client, err := o.KubeClient()
			if err != nil {
				return err
			}
			registry = iks.GetClusterRegistry(client)
			configJSON, err = iks.GetRegistryConfigJSON(registry)
			if err != nil {
				return errors.Wrap(err, "getting IKS registry configuration")
			}

		case cloud.OPENSHIFT, cloud.MINISHIFT:
			registry = "docker-registry.default.svc:5000"
			_, err := o.enableOpenShiftRegistryPermissions(requirements.Cluster.Namespace, registry)
			if err != nil {
				return errors.Wrap(err, "enabling OpenShift registry permissions")
			}
		}

		if registry != "" {
			requirements.Cluster.Registry = registry
			err = requirements.SaveConfig(requirementsFileName)
			if err != nil {
				return errors.Wrapf(err, "failed to save changes to file: %s", requirementsFileName)
			}
		}
	}
	if configJSON != "" {
		// TODO update the secret if its changed
		log.Logger().Warn("jx boot does not yet support automatically populating the container registry secrets automatically...")
	}
	return nil
}

func (o *StepCreateValuesOptions) enableOpenShiftRegistryPermissions(ns string, registry string) (string, error) {
	log.Logger().Infof("Enabling permissions for OpenShift registry in namespace %s", ns)
	// Open the registry so any authenticated user can pull images from the jx namespace
	err := o.RunCommand("oc", "adm", "policy", "add-role-to-group", "system:image-puller", "system:authenticated", "-n", ns)
	if err != nil {
		return "", err
	}
	err = o.EnsureServiceAccount(ns, "jenkins-x-registry")
	if err != nil {
		return "", err
	}
	err = o.RunCommand("oc", "adm", "policy", "add-cluster-role-to-user", "registry-admin", "system:serviceaccount:"+ns+":jenkins-x-registry")
	if err != nil {
		return "", err
	}
	registryToken, err := o.GetCommandOutput("", "oc", "serviceaccounts", "get-token", "jenkins-x-registry", "-n", ns)
	if err != nil {
		return "", err
	}
	return `{"auths": {"` + registry + `": {"auth": "` + base64.StdEncoding.EncodeToString([]byte("serviceaccount:"+registryToken)) + `"}}}`, nil
}
