package spring

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	OptionGroupId        = "group"
	OptionArtifactId     = "artifact"
	OptionLanguage       = "language"
	OptionJavaVersion    = "java-version"
	OptionBootVersion    = "boot-version"
	OptionPackaging      = "packaging"
	OptionDependency     = "dep"
	OptionDependencyKind = "kind"
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
	err := model.ValidateInput(OptionLanguage, &model.Language, data.Language)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionBootVersion, &model.BootVersion, data.BootVersion)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionJavaVersion, &model.JavaVersion, data.JavaVersion)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionPackaging, &model.Packaging, data.Packaging)
	if err != nil {
		return err
	}
	err = model.ValidateTreeInput(OptionDependency, &model.Dependencies, data.Dependencies)
	if err != nil {
		return err
	}

	var qs = []*survey.Question{}
	if data.Language == "" {
		qs = append(qs, CreateValueSelect("Language", "language", &model.Language, data))
	}
	if data.BootVersion == "" && advanced {
		qs = append(qs, CreateValueSelect("Spring Boot version", "bootVersion", &model.BootVersion, data))
	}
	if data.JavaVersion == "" && advanced {
		qs = append(qs, CreateValueSelect("Java version", "javaVersion", &model.JavaVersion, data))
	}
	if data.Packaging == "" && advanced {
		qs = append(qs, CreateValueSelect("Packaging", "packaging", &model.Packaging, data))
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

func (options *SpringOptions) StringArray() []string {
	values := []string{}
	for _, o := range options.Values {
		id := o.ID
		if id != "" {
			values = append(values, id)
		}
	}
	sort.Strings(values)
	return values
}

func (options *SpringTreeSelect) StringArray() []string {
	values := []string{}
	for _, g := range options.Values {
		for _, o := range g.Values {
			id := o.ID
			if id != "" {
				values = append(values, id)
			}
		}
	}
	sort.Strings(values)
	return values
}

func (model *SpringBootModel) ValidateInput(name string, options *SpringOptions, value string) error {
	if value != "" && options != nil {
		for _, v := range options.Values {
			if v.ID == value {
				return nil
			}
		}
		return invalidOption(name, value, options.StringArray())
	}
	return nil
}

func (model *SpringBootModel) ValidateTreeInput(name string, options *SpringTreeSelect, values []string) error {
	if values != nil && len(values) > 0 && options != nil {
		for _, value := range values {
			if value != "" {
				valid := false
				for _, g := range options.Values {
					for _, o := range g.Values {
						if o.ID == value {
							valid = true
							break
						}
					}
				}
				if !valid {
					return invalidOption(name, value, options.StringArray())
				}
			}
		}
	}
	return nil
}

func invalidOption(name string, value string, values []string) error {
	suggestions := util.SuggestionsFor(value, values, util.DefaultSuggestionsMinimumDistance)
	if len(suggestions) > 0 {
		if len(suggestions) == 1 {
			return fmt.Errorf("Invalid option: --%s %s\nDid you mean:  --%s %s", name, value, name, suggestions[0])
		}
		return fmt.Errorf("Invalid option: --%s %s\nDid you mean one of: %s", name, value, strings.Join(suggestions, ", "))
	}
	return fmt.Errorf("Invalid option: --%s %s\nPossible values: %s", name, value, strings.Join(values, ", "))
}

func CreateValueSelect(message string, name string, options *SpringOptions, data *SpringBootForm) *survey.Question {
	values := options.StringArray()
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
