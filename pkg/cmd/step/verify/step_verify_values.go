package verify

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

// StepVerifyValuesOptions contains the command line options
type StepVerifyValuesOptions struct {
	step.StepOptions

	SchemaFile         string
	SchemaTemplateFile string
	RequirementsDir    string
	ValuesFile         string
}

const (
	schemaFileOption      = "schema-file"
	requirementsDirOption = "requirements-dir"
	valuesFileOption      = "values-file"
)

// NewCmdStepVerifyValues constructs the command
func NewCmdStepVerifyValues(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyValuesOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use: "values",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.SchemaFile, schemaFileOption, "s", "", "the path to the schema file")
	cmd.Flags().StringVarP(&options.RequirementsDir, requirementsDirOption, "r", "",
		fmt.Sprintf("the path to the dir which contains the %s file, if omitted looks in the current directory",
			config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&options.ValuesFile, valuesFileOption, "v", "", "the path to the values file")

	return cmd
}

func (o *StepVerifyValuesOptions) checkFile(file string) error {
	if exists, err := util.FileExists(file); !exists || err != nil {
		return fmt.Errorf("provided file %q does not exists", file)
	}
	return nil
}

func (o *StepVerifyValuesOptions) checkOptions() error {
	if o.SchemaFile == "" {
		return util.MissingOption(schemaFileOption)
	}

	if err := o.checkFile(o.SchemaFile); err != nil {
		return err
	}

	if o.RequirementsDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "get current working directory to lookup for %s",
				config.RequirementsConfigFileName)
		}
		o.RequirementsDir = dir
	}

	if exists, err := util.DirExists(o.RequirementsDir); !exists || err != nil {
		return fmt.Errorf("provided dir for requirements does not exist")
	}

	if o.ValuesFile == "" {
		return util.MissingOption(valuesFileOption)
	}

	if err := o.checkFile(o.ValuesFile); err != nil {
		return err
	}

	return nil
}

// Run implements this command
func (o *StepVerifyValuesOptions) Run() error {
	if err := o.checkOptions(); err != nil {
		return err
	}

	requirements, reqFile, err := config.LoadRequirementsConfig(o.RequirementsDir)
	if err != nil {
		return errors.Wrapf(err, "loading requirements from %q", o.RequirementsDir)
	}
	if err := o.checkFile(reqFile); err != nil {
		return err
	}

	schema, err := surveyutils.ReadSchemaTemplate(o.SchemaFile, requirements)
	if err != nil {
		return errors.Wrapf(err, "rendering the schema template %q", o.SchemaFile)
	}

	values, err := ioutil.ReadFile(o.ValuesFile)
	if err != nil {
		return errors.Wrapf(err, "reading the values from file %q", o.ValuesFile)
	}

	values, err = convertYamlToJson(values)
	if err != nil {
		return errors.Wrap(err, "converting values data from YAML to JSON")
	}

	if err := o.verfifySchema(schema, values); err != nil {
		name := filepath.Base(o.ValuesFile)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		log.Logger().Infof(`
The %q values file needs to be updated. You can regnerate the values file from schema %q with command:

jx step create values --name %s
		`, o.ValuesFile, o.SchemaFile, name)
		return errors.Wrap(err, "verifying provided values file against schema file")
	}

	return nil
}

func (o *StepVerifyValuesOptions) verfifySchema(schema []byte, values []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(schema)
	valuesLoader := gojsonschema.NewBytesLoader(values)
	result, err := gojsonschema.Validate(schemaLoader, valuesLoader)
	if err != nil {
		return errors.Wrap(err, "validating the JSON schema against the values")
	}

	if result.Valid() {
		return nil
	}

	var failed bool
	for _, err := range result.Errors() {
		switch err.Type() {
		// The values sometimes contains placeholders to value/local secret path
		// which doesn't satisfy the schema validation.
		case "pattern", "string_gte", "number_gte":
			log.Logger().Warnf("%s", err)
		default:
			log.Logger().Errorf("%s", err)
			failed = true
		}
	}

	if failed {
		return errors.New("invalid values")
	}
	return nil
}

func convertYamlToJson(yml []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yml, &data); err != nil {
		return nil, errors.Wrap(err, "unmarshaling yaml data")
	}

	data = convertType(data)

	result, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling data to json")
	}
	return result, nil
}

func convertType(t interface{}) interface{} {
	switch x := t.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[k.(string)] = convertType(v)
		}
		return m
	case []interface{}:
		for k, v := range x {
			x[k] = convertType(v)
		}
	}
	return t
}
