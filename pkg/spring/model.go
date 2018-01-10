package spring

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
	"os"
)

var (
	DefaultDependencyKinds = []string{"Core", "Web", "Template Engines", "SQL", "I/O", "Ops"}
)

type SpringValue struct {
	Type    string
	Default string
}

type SpringOption struct {
	ID           string
	Name         string
	Description  string
	VersionRange string
}

type SpringOptions struct {
	Type    string
	Default string
	Values  []SpringOption
}

type SpringTreeGroup struct {
	Name   string
	Values []SpringOption
}

type SpringTreeSelect struct {
	Type   string
	Values []SpringTreeGroup
}

type SpringBootModel struct {
	Packaging    SpringOptions
	Language     SpringOptions
	JavaVersion  SpringOptions
	BootVersion  SpringOptions
	GroupId      SpringValue
	ArtifactId   SpringValue
	Version      SpringValue
	Name         SpringValue
	Description  SpringValue
	PackageName  SpringValue
	Dependencies SpringTreeSelect
}

type SpringBootForm struct {
	Packaging       string
	Language        string
	JavaVersion     string
	BootVersion     string
	GroupId         string
	ArtifactId      string
	Version         string
	Name            string
	PackageName     string
	Dependencies    []string
	DependencyKinds []string
}

func LoadSpringBoot() (*SpringBootModel, error) {
	url := "http://start.spring.io"
	spaceClient := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := spaceClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	model := SpringBootModel{}
	err = json.Unmarshal(body, &model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (model *SpringBootModel) CreateSurvey(data *SpringBootForm, advanced bool) error {
	var qs = []*survey.Question{}
	if data.Language == "" {
		qs = append(qs, CreateValueSelect("Language", "language", &model.Language, data))
	}
	if data.BootVersion == "" && advanced {
		qs = append(qs, CreateValueSelect("Spring Boot version", "bootVersion", &model.BootVersion, data))
	}
	if data.GroupId == "" {
		qs = append(qs, CreateValueInput("Group", "groupId", &model.GroupId, data))
	}
	if data.ArtifactId == "" {
		qs = append(qs, CreateValueInput("Artifact", "artifactId", &model.ArtifactId, data))
	}
	if emptyArray(data.Dependencies) {
		qs = append(qs, CreateSpringTreeSelect("Dependencies", "dependencies", &model.Dependencies, data))
	}
	return survey.Ask(qs, data)
}

func CreateValueSelect(message string, name string, options *SpringOptions, data *SpringBootForm) *survey.Question {
	values := []string{}
	for _, o := range options.Values {
		id := o.ID
		if id != "" {
			values = append(values, id)
		}
	}
	sort.Strings(values)
	return &survey.Question{
		Name: name,
		Prompt: &survey.Select{
			Message: message + ":",
			Options: values,
			Default: options.Default,
		},
		Validate: survey.Required,
	}
}

func CreateValueInput(message string, name string, value *SpringValue, data *SpringBootForm) *survey.Question {
	return &survey.Question{
		Name: name,
		Prompt: &survey.Input{
			Message: message + ":",
			Default: value.Default,
		},
		Validate: survey.Required,
	}
}

func CreateSpringTreeSelect(message string, name string, tree *SpringTreeSelect, data *SpringBootForm) *survey.Question {
	dependencyKinds := []string{}
	if data.DependencyKinds != nil {
		dependencyKinds = data.DependencyKinds
	}
	if len(dependencyKinds) == 0 {
		dependencyKinds = DefaultDependencyKinds
	}
	values := []string{}
	for _, t := range tree.Values {
		name := t.Name
		if util.StringArrayIndex(dependencyKinds, name) >= 0 {
			for _, v := range t.Values {
				id := v.ID
				if id != "" {
					values = append(values, id)
				}
			}
		}
	}
	sort.Strings(values)
	return &survey.Question{
		Name: name,
		Prompt: &survey.MultiSelect{
			Message: message + ":",
			Options: values,
		},
		Validate: survey.Required,
	}
}

func (data *SpringBootForm) CreateProject(workDir string) (string, error) {
	dirName := data.ArtifactId
	if dirName == "" {
		dirName = "project"
	}
	answer := filepath.Join(workDir, dirName)

	u := "http://start.spring.io/starter.zip"
	client := http.Client{}

	form := url.Values{}
	data.AddFormValues(&form)

	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return answer, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return answer, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return answer, err
	}

	dir := filepath.Join(workDir, dirName)
	zipFile := dir + ".zip"
	err = ioutil.WriteFile(zipFile, body, util.DefaultWritePermissions)
	if err != nil {
		return answer, fmt.Errorf("Failed to download file %s due to %s", zipFile, err)
	}
	err = util.Unzip(zipFile, dir)
	if err != nil {
		return answer, fmt.Errorf("Failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return answer, err
	}
	return answer, nil
}

func (data *SpringBootForm) AddFormValues(form *url.Values) {
	AddFormValue(form, "packaging", data.Packaging)
	AddFormValue(form, "language", data.Language)
	AddFormValue(form, "javaVersion", data.JavaVersion)
	AddFormValue(form, "bootVersion", data.BootVersion)
	AddFormValue(form, "groupId", data.GroupId)
	AddFormValue(form, "artifactId", data.ArtifactId)
	AddFormValue(form, "version", data.Version)
	AddFormValue(form, "name", data.Name)
	AddFormValues(form, "dependencies", data.Dependencies)
}

func AddFormValues(form *url.Values, key string, values []string) {
	for _, v := range values {
		if v != "" {
			form.Add(key, v)
		}
	}
}

func AddFormValue(form *url.Values, key string, v string) {
	if v != "" {
		form.Add(key, v)
	}
}

func emptyArray(values []string) bool {
	return values == nil || len(values) == 0
}
