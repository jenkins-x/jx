package v1

import (
	"bytes"
	"fmt"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/stoewer/go-strcase"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
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

	Spec ExtensionDetails `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExtensionList is a list of Extension resources
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Extension `json:"items"`
}

// ExtensionDetails containers details of a user
type ExtensionDetails struct {
	Name        string               `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description string               `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	Version     string               `json:"version,omitempty"  protobuf:"bytes,3,opt,name=version"`
	Script      string               `json:"script,omitempty"  protobuf:"bytes,4,opt,name=script"`
	When        []ExtensionWhen      `json:"when,omitempty"  protobuf:"bytes,5,opt,name=when"`
	Given       ExtensionGiven       `json:"given,omitempty"  protobuf:"bytes,6,opt,name=given"`
	Type        ExtensionType        `json:"type,omitempty"  protobuf:"bytes,7,opt,name=type"`
	Parameters  []ExtensionParameter `json:"parameters,omitempty"  protobuf:"bytes,8,opt,name=parameters"`

	// TODO Pre         ExtensionCondition   `json:"pre,omitempty"  protobuf:"bytes,4,opt,name=pre"`
}

type ExtensionWhen string

const (
	// Executed before a pipeline starts
	ExtensionWhenPre ExtensionWhen = "Pre"
	// Executed after a pipeline completes
	ExtensionWhenPost ExtensionWhen = "Post"
	// Executed when an extension installs
	ExtensionWhenInstall ExtensionWhen = "OnInstall"
	// Executed when an extension upgrades
	ExtensionWhenUpgrade ExtensionWhen = "OnUpgrade"
)

type ExtensionGiven string

const (
	ExtensionConditionAlways  ExtensionGiven = "Always"
	ExtensionConditionFailure ExtensionGiven = "Failure"
	ExtensionConditionSuccess ExtensionGiven = "Success"
)

type ExtensionParameter struct {
	Name                    string `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description             string `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	EnvironmentVariableName string `json:"environmentVariableName,omitempty"  protobuf:"bytes,3,opt,name=environmentVariableName"`
	DefaultValue            string `json:"defaultValue,omitempty"  protobuf:"bytes,3,opt,name=defaultValue"`
}

type ExtensionType string

const (
	ExtensionTypeBash ExtensionType = "Bash"
	ExtensionTypeAny  ExtensionType = "Any"
)

type ExecutableExtension struct {
	Name                 string            `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description          string            `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	Script               string            `json:"script,omitempty"  protobuf:"bytes,3,opt,name=script"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty protobuf:"bytes,4,opt,name=environmentvariables"`
	Given                ExtensionGiven    `json:"given,omitempty"  protobuf:"bytes,5,opt,name=given"`
	Type                 ExtensionType     `json:"type,omitempty"  protobuf:"bytes,6,opt,name=type"`
}

func (e *ExecutableExtension) Execute(verbose bool) (err error) {
	scriptFile, err := ioutil.TempFile("", fmt.Sprintf("%s-*", e.Name))
	if err != nil {
		return err
	}
	script := ""
	if e.Type == ExtensionTypeBash || e.Type == "" {
		if !strings.HasPrefix("#!", e.Script) {
			script = fmt.Sprintf("#!/bin/sh\n%s\n", e.Script)
		}
	} else {
		script = e.Script
	}
	_, err = scriptFile.Write([]byte(script))
	if err != nil {
		return err
	}
	err = scriptFile.Chmod(0755)
	if err != nil {
		return err
	}
	if verbose {
		log.Infof("Environment Variables:\n %s\n", e.EnvironmentVariables)
		log.Infof("Script:\n %s\n", script)
	}
	cmd := util.Command{
		Name: scriptFile.Name(),
		Env:  e.EnvironmentVariables,
	}
	log.Infof("Running Extension %s\n", util.ColorInfo(e.Name))
	out, err := cmd.RunWithoutRetry()
	log.Infoln(out)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error executing script %s", e.Name))
	}
	return nil
}

func (e *ExtensionDetails) ToExecutable(envVarValues map[string]string) (ext ExecutableExtension, envVarsStr string, err error) {
	envVars := make(map[string]string)
	for _, p := range e.Parameters {
		value := p.DefaultValue
		if v, ok := envVarValues[p.Name]; ok {
			value = v
		}
		// TODO Log any parameters from RepoExetensions NOT used
		if value != "" {
			envVarName := p.EnvironmentVariableName
			if envVarName == "" {
				envVarName = strings.ToUpper(strcase.SnakeCase(p.Name))
			}
			envVars[envVarName] = value
		}
	}
	res := ExecutableExtension{
		Name:                 e.Name,
		Description:          e.Description,
		Script:               e.Script,
		Given:                e.Given,
		Type:                 e.Type,
		EnvironmentVariables: envVars,
	}
	envVarsFormatted := new(bytes.Buffer)
	for key, value := range envVars {
		fmt.Fprintf(envVarsFormatted, "%s=%s, ", key, value)
	}
	return res, strings.TrimSuffix(envVarsFormatted.String(), ", "), err
}
