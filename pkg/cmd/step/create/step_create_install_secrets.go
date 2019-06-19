package create

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
)

const (
	// SecretsTemplateFileName contains the template used to generate the secrets.yaml file which can be safely checked into git
	SecretsTemplateFileName = "template-secrets.yaml"

	// SecretsParametersFileName the secret parameter files which are never checked into git and may come from, say, Vault
	SecretsParametersFileName = "secrets-parameters.yaml"
)

var (
	createInstallSecretsLong = templates.LongDesc(`
		Creates or updates the install/upgrade secrets.yaml file from a template you can safely check into git: 'template-secrets.yaml' and a local secret parameters file: 'secrets-parameters.yaml'


		It is common for the secrets you need to inject into a variety of charts to share common values so the 'template-secrets.yaml' allows you to define a template to create your 'secrets.yaml' file using a more concise 'secrets-parameters.yaml' file.

		e.g. you may inject your pipeline user name and token from your git provider into various charts like Tekton and Prow using the same values. Or you may want to use template functions to extract values for secrets from specific locations in the parameters file or from Vault locations etc.

		So this templating mechanism lets us compose secret values from input parameters or arbitrary functions in a very similar way to the use of go templates inside YAML files in helm charts themselves.

		You can then use the '.gitignore' rule of 'secrets*.yaml' to avoid ever checking in the 'secrets-parameters.yaml' or the generated 'secrets.yaml' files while safely adding your 'template-secrets.yaml' file.
`)

	createInstallSecretsExample = templates.Examples(`
		# create the secrets.yaml file in the current directory from template-secrets.yaml and secrets-parameters.yaml
		jx step create install secrets	
			`)
)

// StepCreateInstallSecretsOptions contains the command line flags
type StepCreateInstallSecretsOptions struct {
	opts.StepOptions

	Dir       string
	Namespace string
}

// NewCmdStepCreateInstallSecrets Creates a new Command object
func NewCmdStepCreateInstallSecrets(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateInstallSecretsOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "install secrets",
		Short:   "Creates or updates the install/upgrade secrets.yaml file from a template you can check into git 'template-secrets.yaml' and a local secret file 'secrets-parameters.yaml",
		Long:    createInstallSecretsLong,
		Example: createInstallSecretsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the values.yaml file")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to install into. Defaults to $DEPLOY_NAMESPACE if not")

	return cmd
}

// Run implements this command
func (o *StepCreateInstallSecretsOptions) Run() error {
	var err error
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	parametersFile := filepath.Join(o.Dir, SecretsParametersFileName)
	templateFile := filepath.Join(o.Dir, SecretsTemplateFileName)
	secretsFile := filepath.Join(o.Dir, "secrets.yaml")

	parametersExists, err := util.FileExists(templateFile)
	if err != nil {
		return err
	}
	templateExists, err := util.FileExists(templateFile)
	if err != nil {
		return err
	}
	if !templateExists && !parametersExists {
		return nil
	}
	if templateExists && !parametersExists {
		return fmt.Errorf("Has a secret templates file %s but no parameters file %s", templateFile, parametersFile)
	}

	params, err := helm.LoadValuesFile(parametersFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load Secrets parameters: %s", templateFile)
	}

	funcMap := engine.FuncMap()
	funcMap["hashPassword"] = util.HashPassword

	tmpl, err := template.New(SecretsTemplateFileName).Option("missingkey=error").Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		return errors.Wrapf(err, "failed to parse Secrets template: %s", templateFile)
	}

	templateData := map[string]interface{}{
		"Values": chartutil.Values(params),
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return errors.Wrapf(err, "failed to execute Secrets template: %s", templateFile)
	}
	err = ioutil.WriteFile(secretsFile, buf.Bytes(), util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save the secrets file: %s", secretsFile)
	}
	log.Logger().Infof("wrote %s\n", util.ColorInfo(secretsFile))
	return nil
}
