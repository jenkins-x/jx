package surveyutils

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/iancoleman/orderedmap"

	"gopkg.in/AlecAivazis/survey.v1"

	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// Type represents a JSON Schema object type current to https://www.ietf.org/archive/id/draft-handrews-json-schema-validation-01.txt
type Type struct {
	Version              string                `json:"$schema,omitempty"`
	Ref                  string                `json:"$ref,omitempty"`
	MultipleOf           *float64              `json:"multipleOf,omitempty"`
	Maximum              *int                  `json:"maximum,omitempty"`
	ExclusiveMaximum     *int                  `json:"exclusiveMaximum,omitempty"`
	Minimum              *int                  `json:"minimum,omitempty"`
	ExclusiveMinimum     *int                  `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int                  `json:"maxLength,omitempty"`
	MinLength            *int                  `json:"minLength,omitempty"`
	Pattern              *string               `json:"pattern,omitempty"`
	AdditionalItems      *Type                 `json:"additionalItems,omitempty"`
	Items                Items                 `json:"items,omitempty"`
	MaxItems             *int                  `json:"maxItems,omitempty"`
	MinItems             *int                  `json:"minItems,omitempty"`
	UniqueItems          bool                  `json:"uniqueItems,omitempty"`
	MaxProperties        *int                  `json:"maxProperties,omitempty"`
	MinProperties        *int                  `json:"minProperties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Properties           Properties            `json:"properties,omitempty"`
	PatternProperties    map[string]*Type      `json:"patternProperties,omitempty"`
	AdditionalProperties *Type                 `json:"additionalProperties,omitempty"`
	Dependencies         map[string]Dependency `json:"dependencies,omitempty"`
	Enum                 []interface{}         `json:"enum,omitempty"`
	Type                 string                `json:"type,omitempty"`
	AllOf                []*Type               `json:"allOf,omitempty"`
	AnyOf                []*Type               `json:"anyOf,omitempty"`
	OneOf                []*Type               `json:"oneOf,omitempty"`
	Not                  *Type                 `json:"not,omitempty"`
	Definitions          Definitions           `json:"definitions,omitempty"`
	Contains             *Type                 `json:"contains, omitempty"`
	Const                *interface{}          `json:"const, omitempty"`
	PropertyNames        *Type                 `json:"propertyNames, omitempty"`
	Title                string                `json:"title,omitempty"`
	Description          string                `json:"description,omitempty"`
	Default              interface{}           `json:"default,omitempty"`
	Format               *string               `json:"format,omitempty"`
	ContentMediaType     *string               `json:"contentMediaType,omitempty"`
	ContentEncoding      *string               `json:"contentEncoding,omitempty"`
}

// Definitions hold schema definitions.
type Definitions map[string]*Type

// Dependency is either a Type or an array of strings, and so requires special unmarshaling from JSON
type Dependency struct {
	Type  *Type    `json:-`
	Array []string `json:-`
}

// UnmarshalJSON performs unmarshals Dependency from JSON, required as the json field can be one of two types
func (d *Dependency) UnmarshalJSON(b []byte) error {
	if b[0] == '[' {
		return json.Unmarshal(b, &d.Array)
	}
	return json.Unmarshal(b, d.Type)
}

// Items is a either a Type or a array of types, and so requires special unmarshaling from JSON
type Items struct {
	Types []*Type `json:-`
	Type  *Type   `json:-`
}

// Properties is a set of ordered key-value pairs, as it is ordered it requires special marshaling to/from JSON
type Properties struct {
	*orderedmap.OrderedMap
}

// UnmarshalJSON performs custom Unmarshaling for Properties allowing us to preserve order,
// which is not a standard JSON feature
func (p *Properties) UnmarshalJSON(b []byte) error {
	m := orderedmap.New()
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	if p.OrderedMap == nil {
		p.OrderedMap = orderedmap.New()
	}
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		t := Type{}
		om, ok := v.(orderedmap.OrderedMap)
		if !ok {
			return fmt.Errorf("Unable to cast nested data structure to OrderedMap")
		}
		values := make(map[string]interface{}, 0)
		for _, k1 := range om.Keys() {
			v1, _ := om.Get(k1)
			values[k1] = v1
		}
		err := util.ToStructFromMapStringInterface(values, &t)
		if err != nil {
			return err
		}
		p.Set(k, &t)
	}
	return nil
}

// UnmarshalJSON performs unmarshals Items from JSON, required as the json field can be one of two types
func (t *Items) UnmarshalJSON(b []byte) error {
	if b[0] == '[' {
		return json.Unmarshal(b, &t.Types)
	}
	err := json.Unmarshal(b, &t.Type)
	if err != nil {
		return err
	}
	return nil
}

// JSONSchemaOptions are options for generating values from a schema
type JSONSchemaOptions struct {
	CreateSecret func(name string, key string, value string) (*jenkinsv1.ResourceReference, error)
}

// GenerateValues examines the schema in schemaBytes, asks a series of questions using in, out and outErr,
// applying validators, returning a generated json file
func (o *JSONSchemaOptions) GenerateValues(schemaBytes []byte, prefixes []string, in terminal.FileReader,
	out terminal.FileWriter,
	outErr io.Writer) ([]byte, error) {
	t := Type{}
	err := json.Unmarshal(schemaBytes, &t)
	if err != nil {
		return nil, err
	}
	output := orderedmap.New()
	err = o.recurse("", prefixes, make([]string, 0), &t, output, make([]survey.Validator, 0), in, out,
		outErr)
	if err != nil {
		return nil, err
	}
	// move the output up a level
	if root, ok := output.Get(""); ok {
		bytes, err := json.Marshal(root)
		return bytes, err
	}
	return make([]byte, 0), fmt.Errorf("unable to find root element in %v", output)

}

func (o *JSONSchemaOptions) recurse(name string, prefixes []string, requiredFields []string, t *Type, output *orderedmap.OrderedMap,
	additionalValidators []survey.Validator, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) error {
	required := util.Contains(requiredFields, name)
	if name != "" {
		prefixes = append(prefixes, name)
	}
	if t.ContentEncoding != nil {
		return fmt.Errorf("contentEncoding is not supported for %s", name)
		// TODO support contentEncoding
	}
	if t.ContentMediaType != nil {
		return fmt.Errorf("contentMediaType is not supported for %s", name)
		// TODO support contentMediaType
	}

	switch t.Type {
	case "null":
		output.Set(name, nil)
	case "boolean":
		validators := []survey.Validator{
			EnumValidator(t.Enum),
			RequiredValidator(required),
			BoolValidator(),
		}
		validators = append(validators, additionalValidators...)
		err := o.handleBasicProperty(name, prefixes, validators, t, output, in, out, outErr)
		if err != nil {
			return err
		}
	case "object":
		result := orderedmap.New()
		if t.AdditionalProperties != nil {
			return fmt.Errorf("additionalProperties is not supported for %s", name)
			// TODO support additionalProperties
		}
		if len(t.PatternProperties) > 0 {
			return fmt.Errorf("patternProperties is not supported for %s", name)
			// TODO support patternProperties
		}
		if len(t.Dependencies) > 0 {
			return fmt.Errorf("dependencies is not supported for %s", name)
			// TODO support dependencies
		}
		if t.PropertyNames != nil {
			return fmt.Errorf("propertyNames is not supported for %s", name)
			// TODO support propertyNames
		}
		if t.Const != nil {
			return fmt.Errorf("const is not supported for %s", name)
			// TODO support const
		}
		duringValidators := []survey.Validator{
			// These validators are run during processing of the properties
			MaxPropertiesValidator(t.MaxProperties, result),
		}
		postValidators := []survey.Validator{
			// These validators are run after the processing of the properties
			MinPropertiesValidator(t.MinProperties, result),
			EnumValidator(t.Enum),
		}
		for valid := false; !valid; {
			for _, n := range t.Properties.Keys() {
				v, _ := t.Properties.Get(n)
				property := v.(*Type)
				err := o.recurse(n, prefixes, t.Required, property, result, duringValidators, in, out, outErr)
				if err != nil {
					return err
				}
			}
			valid = true
			for _, v := range postValidators {
				err := v(result)
				if err != nil {
					log.Errorf("%v\n", err.Error())
					valid = false
				}
			}
		}

		output.Set(name, result)
	case "array":
		if t.Const != nil {
			return fmt.Errorf("const is not supported for %s", name)
			// TODO support const
		}
		if t.Contains != nil {
			return fmt.Errorf("contains is not supported for %s", name)
			// TODO support contains
		}
		if t.AdditionalItems != nil {
			return fmt.Errorf("additionalItems is not supported for %s", name)
			// TODO support additonalItems
		}
		err := o.handleArrayProperty(name, t, output, in, out, outErr)
		if err != nil {
			return err
		}
	case "number":
		validators := additionalValidators
		validators = append(validators, FloatValidator())
		err := o.handleBasicProperty(name, prefixes, numberValidator(required, validators, t), t, output, in, out,
			outErr)
		if err != nil {
			return err
		}
	case "string":
		validators := []survey.Validator{
			EnumValidator(t.Enum),
			MinLengthValidator(t.MinLength),
			MaxLengthValidator(t.MaxLength),
			RequiredValidator(required),
			PatternValidator(t.Pattern),
		}
		// Custom format support for password

		// Defined Format validation
		if t.Format != nil {
			format := util.DereferenceString(t.Format)
			switch format {
			case "date-time":
				validators = append(validators, DateTimeValidator())
			case "date":
				validators = append(validators, DateValidator())
			case "time":
				validators = append(validators, TimeValidator())
			case "email":
				validators = append(validators, EmailValidator())
			case "idn-email":
				validators = append(validators, EmailValidator())
			case "hostname":
				validators = append(validators, HostnameValidator())
			case "idn-hostname":
				validators = append(validators, HostnameValidator())
			case "ipv4":
				validators = append(validators, Ipv4Validator())
			case "ipv6":
				validators = append(validators, Ipv6Validator())
			case "uri":
				validators = append(validators, URIValidator())
			case "uri-reference":
				validators = append(validators, URIReferenceValidator())
			case "iri":
				return fmt.Errorf("iri defined format not supported")
			case "iri-reference":
				return fmt.Errorf("iri-reference defined format not supported")
			case "uri-template":
				return fmt.Errorf("uri-template defined format not supported")
			case "json-pointer":
				validators = append(validators, JSONPointerValidator())
			case "relative-json-pointer":
				return fmt.Errorf("relative-json-pointer defined format not supported")
			case "regex":
				return fmt.Errorf("regex defined format not supported, use pattern keyword")
			}
		}
		validators = append(validators, additionalValidators...)
		err := o.handleBasicProperty(name, prefixes, validators, t, output, in, out, outErr)
		if err != nil {
			return err
		}
	case "integer":
		validators := additionalValidators
		validators = append(validators, IntegerValidator())
		err := o.handleBasicProperty(name, prefixes, numberValidator(required, validators, t), t, output, in, out,
			outErr)
		if err != nil {
			return err
		}
	}

	return nil
}

// According to the spec, "An instance validates successfully against this keyword if its value
// is equal to the value of the keyword." which we interpret for questions as "this is the value of this keyword"
func (o *JSONSchemaOptions) handleConst(name string, validators []survey.Validator, t *Type, output *orderedmap.OrderedMap, in terminal.FileReader,
	out terminal.FileWriter, outErr io.Writer) error {
	message := fmt.Sprintf("Do you want to set %s to %v", name, *t.Const)
	help := ""
	if t.Title != "" {
		message = t.Title
	}
	if t.Description != "" {
		help = t.Description
	}
	prompt := &survey.Confirm{
		Help:    help,
		Message: message,
		Default: true,
	}
	answer := false
	surveyOpts := survey.WithStdio(in, out, outErr)
	validator := survey.ComposeValidators(validators...)

	err := survey.AskOne(prompt, &answer, NoopValidator(), surveyOpts)
	if err != nil {
		return err
	}
	err = validator(t.Const)
	if answer {
		output.Set(name, t.Const)
	}
	return nil
}

func (o *JSONSchemaOptions) handleArrayProperty(name string, t *Type, output *orderedmap.OrderedMap, in terminal.FileReader,
	out terminal.FileWriter, outErr io.Writer) error {
	results := make([]interface{}, 0)

	validators := []survey.Validator{
		MaxItemsValidator(t.MaxItems, results),
		UniqueItemsValidator(results),
		MinItemsValidator(t.MinItems, results),
		EnumValidator(t.Enum),
	}
	// Normally arrays are used to create a multi-select list
	// Note that this only supports basic types at the moment
	if t.Items.Type != nil && t.Items.Type.Enum != nil {
		if t.Items.Type.Type == "null" {
			output.Set(name, nil)
			return nil
		} else if !util.Contains([]string{"string", "boolean", "number", "integer"}, t.Items.Type.Type) {
			return fmt.Errorf("type %s is not supported for array %s", t.Items.Type.Type, name)
			// TODO support other types
		}
		message := fmt.Sprintf("Select values for %s", name)
		help := ""
		var defaultValue []string
		options := make([]string, 0)
		if t.Title != "" {
			message = t.Title
		}
		if t.Description != "" {
			help = t.Description
		}
		for _, e := range t.Items.Type.Enum {
			options = append(options, fmt.Sprintf("%v", e))
		}
		if t.Default != nil {
			defaultString, err := util.AsString(t.Default)
			defaultArray, err1 := util.AsSliceOfStrings(t.Default)
			if err != nil && err1 != nil {
				v := reflect.ValueOf(t.Default)
				v = reflect.Indirect(v)
				return fmt.Errorf("Cannot convert %v (%v) to string or []string", v.Type(), t.Default)
			}
			if defaultString != "" {
				defaultValue = []string{
					defaultString,
				}
			} else {
				defaultValue = defaultArray
			}
		}
		answer := make([]string, 0)
		surveyOpts := survey.WithStdio(in, out, outErr)
		// TODO^^^ Apply correct validators for type
		validator := survey.ComposeValidators(validators...)

		prompt := &survey.MultiSelect{
			Default: defaultValue,
			Help:    help,
			Message: message,
			Options: options,
		}
		err := survey.AskOne(prompt, &answer, validator, surveyOpts)
		if err != nil {
			return err
		}
		for _, a := range answer {
			v, err := convertAnswer(a, t.Items.Type.Type)
			// An error is a genuine error as we've already done type validation
			if err != nil {
				return err
			}
			results = append(results, v)
		}
	}

	output.Set(name, results)
	return nil
}

func convertAnswer(answer string, t string) (interface{}, error) {
	if t == "number" {
		return strconv.ParseFloat(answer, 64)
	} else if t == "integer" {
		return strconv.Atoi(answer)
	} else if t == "boolean" {
		return strconv.ParseBool(answer)
	} else {
		return answer, nil
	}
}

func (o *JSONSchemaOptions) handleBasicProperty(name string, prefixes []string, validators []survey.Validator, t *Type,
	output *orderedmap.OrderedMap, in terminal.FileReader,
	out terminal.FileWriter, outErr io.Writer) error {
	if t.Const != nil {
		return o.handleConst(name, validators, t, output, in, out, outErr)
	}
	message := fmt.Sprintf("Enter a value for %s", name)
	help := ""
	defaultValue := ""
	if t.Title != "" {
		message = t.Title
	}
	if t.Description != "" {
		help = t.Description
	}
	if t.Default != nil {
		defaultValue = fmt.Sprintf("%v", t.Default)
	}
	answer := ""
	surveyOpts := survey.WithStdio(in, out, outErr)
	validator := survey.ComposeValidators(validators...)
	// Ask the question
	// Custom format support for passwords
	storeAsSecret := false
	if util.DereferenceString(t.Format) == "password" || util.DereferenceString(t.Format) == "token" {
		storeAsSecret = true
		// Basic input
		prompt := &survey.Password{
			Message: message,
			Help:    help,
		}

		err := survey.AskOne(prompt, &answer, validator, surveyOpts)
		if err != nil {
			return err
		}
	} else if t.Enum != nil {
		// Support for selects
		names := make([]string, 0)
		for _, e := range t.Enum {
			names = append(names, fmt.Sprintf("%v", e))
		}
		prompt := &survey.Select{
			Message: message,
			Options: names,
			Default: defaultValue,
			Help:    help,
		}
		err := survey.AskOne(prompt, &answer, validator, surveyOpts)
		if err != nil {
			return err
		}
	} else {
		// Basic input
		prompt := &survey.Input{
			Message: message,
			Default: defaultValue,
			Help:    help,
		}

		err := survey.AskOne(prompt, &answer, validator, surveyOpts)
		if err != nil {
			return err
		}
	}

	v, err := convertAnswer(answer, t.Type)
	if err != nil {
		return err
	}
	if storeAsSecret {
		secretName := kube.ToValidName(strings.Join(append(prefixes, "secret"), "-"))
		value, err := util.AsString(v)
		if err != nil {
			return err
		}
		secretReference, err := o.CreateSecret(secretName, util.DereferenceString(t.Format), value)
		if err != nil {
			return err
		}
		output.Set(name, secretReference)
	} else {
		// Write the value to the output
		output.Set(name, v)
	}
	return nil
}

// integers and numbers validate identically, but we have to repeat ourselves as Go has no generics
func numberValidator(required bool, additonalValidators []survey.Validator, t *Type) []survey.Validator {
	validators := []survey.Validator{EnumValidator(t.Enum),
		MultipleOfValidator(t.MultipleOf),
		RequiredValidator(required),
		MinValidator(t.MinLength, false),
		MinValidator(t.ExclusiveMinimum, true),
		MaxValidator(t.Maximum, false),
		MaxValidator(t.ExclusiveMaximum, true),
	}
	return append(validators, additonalValidators...)
}
