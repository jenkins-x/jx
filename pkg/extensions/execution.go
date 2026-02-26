package extensions

import (
	"bytes"
	"fmt"
	"strings"

	jenkinsv1client "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghodss/yaml"
	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/stoewer/go-strcase"
)

// TODO remove the env vars formatting stuff from here and make it a function on ExtensionSpec
func ToExecutable(e *jenkinsv1.ExtensionSpec, paramValues []jenkinsv1.ExtensionParameterValue, teamNamespace string, exts jenkinsv1client.ExtensionInterface) (ext jenkinsv1.ExtensionExecution, envVarsStr string, err error) {
	envVars := make([]jenkinsv1.EnvironmentVariable, 0)
	paramValueLookup := make(map[string]string, 0)
	for _, v := range paramValues {
		paramValueLookup[v.Name] = v.Value
	}
	for _, p := range e.Parameters {
		value := p.DefaultValue
		if v, ok := paramValueLookup[p.Name]; ok {
			value = v
		}
		// TODO Log any parameters from RepoExetensions NOT used
		if value != "" {
			envVarName := p.EnvironmentVariableName
			if envVarName == "" {
				envVarName = namespaceName(e.Namespace, e.Name, p.Name)
			}
			envVars = append(envVars, jenkinsv1.EnvironmentVariable{
				Name:  envVarName,
				Value: value,
			})
		}
	}

	extension, err := exts.Get(e.FullyQualifiedKebabName(), metav1.GetOptions{})
	if err != nil {
		return jenkinsv1.ExtensionExecution{}, "", fmt.Errorf("Unable to find extension definition %s. %v", e.FullyQualifiedKebabName(), err)
	}
	// Create an owner ref yaml snippet for this extension
	ownerRef, err := yaml.Marshal(kube.ExtensionOwnerRef(extension))
	if err != nil {
		return jenkinsv1.ExtensionExecution{}, "", err
	}

	// Add Global vars
	envVars = append(envVars,
		jenkinsv1.EnvironmentVariable{
			Name:  namespaceName(jenkinsv1.VersionGlobalParameterName),
			Value: e.Version,
		}, jenkinsv1.EnvironmentVariable{
			Name:  namespaceName(jenkinsv1.TeamNamespaceGlobalParameterName),
			Value: teamNamespace,
		},
		jenkinsv1.EnvironmentVariable{
			Name:  namespaceName(jenkinsv1.OwnerReferenceGlobalParameterName),
			Value: string(ownerRef),
		},
	)
	res := jenkinsv1.ExtensionExecution{
		Name:                 e.Name,
		Namespace:            e.Namespace,
		UUID:                 e.UUID,
		Description:          e.Description,
		Script:               e.Script,
		Given:                e.Given,
		EnvironmentVariables: envVars,
	}
	envVarsFormatted := new(bytes.Buffer)
	for _, envVar := range envVars {
		fmt.Fprintf(envVarsFormatted, "%s=%s, ", envVar.Name, envVar.Value)
	}
	return res, strings.TrimSuffix(envVarsFormatted.String(), ", "), err
}

func namespaceName(names ...string) string {
	format := strings.TrimPrefix(strings.Repeat("_%s", len(names)), "_")
	vars := make([]interface{}, 0)
	for _, a := range names {
		vars = append(vars, strings.ToUpper(strcase.SnakeCase(a)))
	}
	return fmt.Sprintf(format, vars...)
}
