package util

import (
	schemagen "github.com/abayer/jsonschema"
	"github.com/xeipuuv/gojsonschema"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// ValidateYaml generates a JSON schema for the given struct type, and then validates the given YAML against that
// schema, ignoring Containers and missing fields.
func ValidateYaml(target interface{}, data []byte) ([]string, error) {
	reflector := schemagen.Reflector{
		IgnoredTypes: []interface{}{
			corev1.Container{},
		},
		RequiredFromJSONSchemaTags: true,
	}
	schema := reflector.Reflect(target)

	dataAsJSON, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}
	schemaLoader := gojsonschema.NewGoLoader(schema)
	documentLoader := gojsonschema.NewBytesLoader(dataAsJSON)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}
	if !result.Valid() {
		errMsgs := []string{}
		for _, e := range result.Errors() {
			errMsgs = append(errMsgs, e.String())
		}
		return errMsgs, nil
	}

	return nil, nil
}
