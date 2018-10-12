package v1

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/stoewer/go-strcase"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Extension represents an extension available to this Jenkins X install
type Extension struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ExtensionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExtensionList is a list of Extensions available for a team
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Extension `json:"items"`
}

// ExtensionSpec provides details of an extension available for a team
type ExtensionSpec struct {
	Name        string               `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description string               `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	Version     string               `json:"version,omitempty"  protobuf:"bytes,3,opt,name=version"`
	Script      string               `json:"script,omitempty"  protobuf:"bytes,4,opt,name=script"`
	When        []ExtensionWhen      `json:"when,omitempty"  protobuf:"bytes,5,opt,name=when"`
	Given       ExtensionGiven       `json:"given,omitempty"  protobuf:"bytes,6,opt,name=given"`
	Parameters  []ExtensionParameter `json:"parameters,omitempty"  protobuf:"bytes,8,opt,name=parameters"`
	Namespace   string               `json:"namespace,omitempty"  protobuf:"bytes,9,opt,name=namespace"`
	UUID        string               `json:"uuid,omitempty"  protobuf:"bytes,10,opt,name=uuid"`
	Children    []string             `json:"children,omitempty"  protobuf:"bytes,11,opt,name=children"`
}

// ExtensionWhen specifies when in the lifecycle an extension should execute. By default Post.
type ExtensionWhen string

const (
	// Executed before a pipeline starts
	ExtensionWhenPre ExtensionWhen = "pre"
	// Executed after a pipeline completes
	ExtensionWhenPost ExtensionWhen = "post"
	// Executed when an extension installs
	ExtensionWhenInstall ExtensionWhen = "onInstall"
	// Executed when an extension upgrades
	ExtensionWhenUpgrade ExtensionWhen = "onUpgrade"
)

// ExtensionGiven specifies the condition (if the extension is executing in a pipeline on which the extension should execute. By default Always.
type ExtensionGiven string

const (
	ExtensionGivenAlways  ExtensionGiven = "Always"
	ExtensionGivenFailure ExtensionGiven = "Failure"
	ExtensionGivenSuccess ExtensionGiven = "Success"
)

// ExtensionParameter describes a parameter definition for an extension
type ExtensionParameter struct {
	Name                    string `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description             string `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	EnvironmentVariableName string `json:"environmentVariableName,omitempty"  protobuf:"bytes,3,opt,name=environmentVariableName"`
	DefaultValue            string `json:"defaultValue,omitempty"  protobuf:"bytes,3,opt,name=defaultValue"`
}

// ExtensionExecution is an executable instance of an extension which can be attached into a pipeline for later execution.
// It differs from an Extension as it cannot have children and parameters have been resolved to environment variables
type ExtensionExecution struct {
	Name                 string                `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description          string                `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	Script               string                `json:"script,omitempty"  protobuf:"bytes,3,opt,name=script"`
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables,omitempty protobuf:"bytes,4,opt,name=environmentvariables"`
	Given                ExtensionGiven        `json:"given,omitempty"  protobuf:"bytes,5,opt,name=given"`
	Namespace            string                `json:"namespace,omitempty"  protobuf:"bytes,7,opt,name=namespace"`
	UUID                 string                `json:"uuid,omitempty"  protobuf:"bytes,8,opt,name=uuid"`
}

// ExtensionRepositoryLockList contains a list of ExtensionRepositoryLock items
type ExtensionRepositoryLockList struct {
	Version    int             `json:"version"`
	Extensions []ExtensionSpec `json:"extensions"`
}

// ExtensionDefinitionReferenceList contains a list of ExtensionRepository items
type ExtensionDefinitionReferenceList struct {
	Remotes []ExtensionDefinitionReference `json:"remotes"`
}

// ExtensionRepositoryReference references a GitHub repo that contains extension definitions
type ExtensionDefinitionReference struct {
	Remote string `json:"remote"`
	Tag    string `json:"tag"`
}

// ExtensionDefinitionList contains a list of ExtensionDefinition items
type ExtensionDefinitionList struct {
	Version    string                `json:"version,omitempty"`
	Extensions []ExtensionDefinition `json:"extensions"`
}

// ExtensionDefinition defines an Extension
type ExtensionDefinition struct {
	Name        string                              `json:"name"`
	Namespace   string                              `json:"namespace"`
	UUID        string                              `json:"uuid"`
	Description string                              `json:"description,omitempty"`
	When        []ExtensionWhen                     `json:"when,omitempty"`
	Given       ExtensionGiven                      `json:"given,omitempty"`
	Children    []ExtensionDefinitionChildReference `json:"children,omitempty"`
	ScriptFile  string                              `json:"scriptFile,omitempty"`
	Parameters  []ExtensionParameter                `json:"parameters,omitempty"`
}

// ExtensionDefinitionChildReference provides a reference to a child
type ExtensionDefinitionChildReference struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Remote    string `json:"remote,omitempty"`
	Tag       string `json:"tag,omitempty"`
}

type EnvironmentVariable struct {
	Name  string `json:"name,omitempty protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value,omitempty protobuf:"bytes,2,opt,name=value"`
}

// ExtensionsConfigList contains a list of ExtensionConfig items
type ExtensionConfigList struct {
	Extensions []ExtensionConfig `json:"extensions"`
}

// ExtensionConfig is the configuration and enablement for an extension inside an app
type ExtensionConfig struct {
	Name       string                    `json:"name"`
	Namespace  string                    `json:"namespace"`
	Parameters []ExtensionParameterValue `json: "parameters"`
}

type ExtensionParameterValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	VersionGlobalParameterName       string = "extVersion"
	TeamNamespaceGlobalParameterName string = "extTeamNamespace"
)

func (e *ExtensionExecution) Execute(verbose bool) (err error) {
	scriptFile, err := ioutil.TempFile("", fmt.Sprintf("%s-*", e.Name))
	if err != nil {
		return err
	}
	_, err = scriptFile.Write([]byte(e.Script))
	if err != nil {
		return err
	}
	err = scriptFile.Chmod(0755)
	if err != nil {
		return err
	}
	err = scriptFile.Close()
	if err != nil {
		return err
	}
	if verbose {
		log.Infof("Environment Variables:\n %s\n", e.EnvironmentVariables)
		log.Infof("Script:\n %s\n", e.Script)
	}
	envVars := make(map[string]string, 0)
	for _, v := range e.EnvironmentVariables {
		envVars[v.Name] = v.Value
	}
	cmd := util.Command{
		Name: scriptFile.Name(),
		Env:  envVars,
	}
	out, err := cmd.RunWithoutRetry()
	log.Infoln(out)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error executing script %s", e.Name))
	}
	return nil
}

// TODO remove the env vars formatting stuff from here and make it a function on ExtensionSpec
func (e *ExtensionSpec) ToExecutable(paramValues []ExtensionParameterValue, teamNamespace string) (ext ExtensionExecution, envVarsStr string, err error) {
	envVars := make([]EnvironmentVariable, 0)
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
				envVarName = e.EnvVarName(e.Namespace, e.Name, p.Name)
			}
			envVars = append(envVars, EnvironmentVariable{
				Name:  envVarName,
				Value: value,
			})
		}
	}
	// Add Global vars
	envVars = append(envVars, EnvironmentVariable{
		Name:  e.EnvVarName(VersionGlobalParameterName),
		Value: e.Version,
	}, EnvironmentVariable{
		Name:  e.EnvVarName(TeamNamespaceGlobalParameterName),
		Value: teamNamespace,
	})
	res := ExtensionExecution{
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

func (e *ExtensionSpec) EnvVarName(names ...string) string {
	format := strings.TrimPrefix(strings.Repeat("_%s", len(names)), "_")
	vars := make([]interface{}, 0)
	for _, a := range names {
		vars = append(vars, strings.ToUpper(strcase.SnakeCase(a)))
	}
	return fmt.Sprintf(format, vars...)
}

func (constraints *ExtensionDefinitionReferenceList) LoadFromFile(inputFile string) (err error) {
	path, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}
	y, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(y, constraints)
	if err != nil {
		return err
	}
	return nil
}

func (lock *ExtensionRepositoryLockList) LoadFromFile(inputFile string) (err error) {
	path, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}
	y, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(y, lock)
	if err != nil {
		return err
	}
	return nil
}

func (lock *ExtensionDefinitionList) LoadFromURL(definitionsUrl string, extension string, version string) (err error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(fmt.Sprintf("%s?version=%d", definitionsUrl, time.Now().UnixNano()/int64(time.Millisecond)))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Infof("Unable to find Extension Definitions at %s for %s with version %s\n", util.ColorWarning(definitionsUrl), util.ColorWarning(extension), util.ColorWarning(version))
		return nil
	}
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, lock)
	if err != nil {
		return err
	}
	return nil
}

func (e *ExtensionDefinition) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionDefinition) FullyQualifiedKebabName() (fqn string) {
	return fmt.Sprintf("%s.%s", strcase.KebabCase(e.Namespace), strcase.KebabCase(e.Name))
}

func (e *ExtensionDefinitionChildReference) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionDefinitionChildReference) FullyQualifiedKebabName() (fqn string) {
	return fmt.Sprintf("%s.%s", strcase.KebabCase(e.Namespace), strcase.KebabCase(e.Name))
}

func (e *ExtensionSpec) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionSpec) FullyQualifiedKebabName() (fqn string) {
	return fmt.Sprintf("%s.%s", strcase.KebabCase(e.Namespace), strcase.KebabCase(e.Name))
}

func (e *ExtensionConfig) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionConfig) FullyQualifiedKebabName() (fqn string) {
	return fmt.Sprintf("%s.%s", strcase.KebabCase(e.Namespace), strcase.KebabCase(e.Name))
}

func (e *ExtensionExecution) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionExecution) FullyQualifiedKebabName() (fqn string) {
	return fmt.Sprintf("%s.%s", strcase.KebabCase(e.Namespace), strcase.KebabCase(e.Name))
}

func (extensionsConfig *ExtensionConfigList) LoadFromFile(inputFile string) (cfg *ExtensionConfigList, err error) {
	extensionsYamlPath, err := filepath.Abs(inputFile)
	if err != nil {
		return nil, err
	}
	extensionsYaml, err := ioutil.ReadFile(extensionsYamlPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(extensionsYaml, extensionsConfig)
	if err != nil {
		return nil, err
	}
	return extensionsConfig, nil
}

func (extensionsConfig *ExtensionConfigList) LoadFromConfigMap(configMapName string, client kubernetes.Interface, namespace string) (cfg *ExtensionConfigList, err error) {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		// CM doesn't exist, create it
		cm, err = client.CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: configMapName,
			},
		})
		if err != nil {
			return nil, err
		}
	}
	extensionsConfig.Extensions = make([]ExtensionConfig, 0)

	extensionConfigList := ExtensionConfigList{}
	err = yaml.Unmarshal([]byte(cm.Data["extensions"]), &extensionConfigList.Extensions)
	if err != nil {
		return nil, err
	}
	return &extensionConfigList, nil
}

func (e *ExtensionSpec) IsPost() bool {
	return e.Contains(e.When, ExtensionWhenPost) || len(e.When) == 0
}

func (e *ExtensionSpec) Contains(whens []ExtensionWhen, when ExtensionWhen) bool {
	for _, w := range whens {
		if when == w {
			return true
		}
	}
	return false
}
