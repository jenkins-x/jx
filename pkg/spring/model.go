package spring

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"gopkg.in/AlecAivazis/survey.v1"
	"github.com/jenkins-x/jx/pkg/util"
	"sort"
)

var (
	defaultDependencyKinds = []string{"Core", "Web", "Template Engines", "SQL", "I/O", "Ops"}
)
type SpringValue struct {
	Type    string
	Default string
}

type SpringOption struct {
	ID   string
	Name string
	Description  string
	VersionRange string
}

type SpringOptions struct {
	Type         string
	Default      string
	Values       []SpringOption
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
	Packaging    string
	Language     string
	JavaVersion  string
	BootVersion  string
	GroupId      string
	ArtifactId   string
	Version      string
	Name         string
	PackageName  string
	Dependencies []string
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
	var qs = []*survey.Question{
		CreateSpringTreeSelect("Dependencies", "dependencies", &model.Dependencies, data),
		CreateValueSelect("Language", "language", &model.Language, data),
		CreateValueSelect("Spring Boot version", "bootVersion", &model.BootVersion, data),
		CreateValueInput("Group", "groupId", &model.GroupId, data),
		CreateValueInput("Artifact", "artifactId", &model.ArtifactId, data),
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
		dependencyKinds = defaultDependencyKinds
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
