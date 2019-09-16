package create

import (
	"bytes"
	"io/ioutil"
	"text/template"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	createTemplatedConfigLong = templates.LongDesc(`
		Creates a YAML config file from a Go template file and a jx requirements file
`)

	createTemplatedConfigExample = templates.Examples(`
		# creates a config file from a template file and a jx requirements file
		jx step create templated config -t mytemplate.tmpl.yml -r jx-requirements.yml -c config.yml
`)
)

const valuesTemplateName = "values-template"

// StepCreateTemplatedConfigOptions command line flags
type StepCreateTemplatedConfigOptions struct {
	step.StepOptions

	TemplateFile    string
	RequirementsDir string
	ConfigFile      string
}

// NewCmdStepCreateTemplatedConfig Creates a new Command object
func NewCmdStepCreateTemplatedConfig(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateTemplatedConfigOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "templated config",
		Short:   "Create a YAML config file from a Go template file and a jx requirements file",
		Long:    createTemplatedConfigLong,
		Example: createTemplatedConfigExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.TemplateFile, "template-file", "t", "", "The template file used to render the config YAML file")
	cmd.Flags().StringVarP(&options.RequirementsDir, "requirements-dir", "r", ".", "The jx requirements file directory")
	cmd.Flags().StringVarP(&options.ConfigFile, "config-file", "c", "", "The rendered config YAML file")

	return cmd
}

// Run implements this command
func (o *StepCreateTemplatedConfigOptions) Run() error {
	requirements, _, err := config.LoadRequirementsConfig(o.RequirementsDir)
	if err != nil {
		return errors.Wrapf(err, "loading requirements file form dir %q", o.RequirementsDir)
	}
	data, err := o.renderTemplate(requirements)
	if err != nil {
		return errors.Wrapf(err, "rendering the config template using the requirements from dir %q", o.RequirementsDir)
	}
	if err := ioutil.WriteFile(o.ConfigFile, data, util.DefaultFileWritePermissions); err != nil {
		return errors.Wrapf(err, "writing the rendered config into file %q", o.ConfigFile)
	}

	log.Logger().Infof("Saved the rendered configuration into %s file", o.ConfigFile)

	return nil
}

func (o *StepCreateTemplatedConfigOptions) renderTemplate(requirements *config.RequirementsConfig) ([]byte, error) {
	tmpl, err := template.New(valuesTemplateName).Option("missingkey=error").ParseFiles(o.TemplateFile)
	if err != nil {
		return nil, errors.Wrap(err, "parsing the template file")
	}
	requirementsMap, err := requirements.ToMap()
	if err != nil {
		return nil, errors.Wrapf(err, "converting requirements into a map: %v", requirements)
	}
	tmplData := map[string]interface{}{
		"Requirements": requirementsMap,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplData)
	if err != nil {
		return nil, errors.Wrapf(err, "rendering the template file: %s", o.TemplateFile)
	}
	return buf.Bytes(), nil
}
