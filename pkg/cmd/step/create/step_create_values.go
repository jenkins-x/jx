package create

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/prometheus/common/log"

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
	opts.StepCreateOptions

	Dir      string
	Name     string
	BasePath string

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
		StepCreateOptions: opts.StepCreateOptions{
			StepOptions: opts.StepOptions{
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
	cmd.Flags().StringVarP(&options.Schema, "schema", "", "", "the path to the schema file, overrides --dir and --name")
	cmd.Flags().StringVarP(&options.Name, "name", "", "values", "the kind of the file to create (and, by default, the schema name)")
	cmd.Flags().StringVarP(&options.BasePath, "secret-base-path", "", "", fmt.Sprintf("the secret path used to store secrets in vault / file system. Typically a unique name per cluster+team. If none is specified we will default it to the cluster name from the %s file in the current or a parent directory.", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&options.ValuesFile, "out", "", "", "the path to the file to create, overrides --dir and --name")
	cmd.Flags().StringVarP(&options.SecretsScheme, optionSecretsScheme, "", "", fmt.Sprintf("the scheme to store/reference any secrets in, valid options are vault and local. If none are specified we will default it from the %s file in the current or a parent directory.", config.RequirementsConfigFileName))
	return cmd
}

// Run implements this command
func (o *StepCreateValuesOptions) Run() error {
	var err error
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
			log.Infof("defaulting to secret storage scheme %s found from requirements file at %s\n", info(o.SecretsScheme), info(fileName))
		} else {
			log.Warnf("there is no requirements file at %s\n", fileName)
		}
	}
	if o.BasePath == "" {
		if exists {
			o.BasePath = string(requirements.ClusterName)
			log.Infof("defaulting to secret base path to the cluster name %s found from requirements file at %s\n", info(o.BasePath), info(fileName))
		} else {
			log.Warnf("there is no requirements file at %s\n", fileName)
		}

	}
	if !(o.SecretsScheme == "vault" || o.SecretsScheme == "local") {
		util.InvalidArgf(optionSecretsScheme, "Use one of vault or local")
	}
	if o.Schema == "" {
		o.Schema = filepath.Join(o.Dir, fmt.Sprintf("%s.schema.json", o.Name))
	}
	if o.ValuesFile == "" {
		o.ValuesFile = filepath.Join(o.Dir, fmt.Sprintf("%s.yaml", o.Name))
	}
	fmt.Println()
	err = o.CreateValuesFile()
	if err != nil {
		return errors.WithStack(err)
	}
	fmt.Println()
	return nil
}

// CreateValuesFile builds the clients and settings from the team needed to run apps.ProcessValues, and then copies the answer
// to the location requested by the command
func (o *StepCreateValuesOptions) CreateValuesFile() error {
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
	secretURLClient, err := o.GetSecretURLClient()
	if err != nil {
		return err
	}
	existing, err := helm.LoadValuesFile(o.ValuesFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load values file %s", o.ValuesFile)
	}

	valuesFileName, cleanup, err := apps.ProcessValues(schema, o.Name, gitOpsURL, teamName, o.BasePath, o.BatchMode, false, secretURLClient, existing, o.SecretsScheme, o.In, o.Out, o.Err, o.Verbose)
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
