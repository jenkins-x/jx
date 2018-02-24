package tmplh

import (
	"bytes"
	tmpl "html/template"

	"github.com/Masterminds/sprig"

	"github.com/antham/chyle/chyle/errh"
)

var store = map[string]interface{}{}

func isset(key string) bool {
	_, ok := store[key]

	return ok
}

func set(key string, value interface{}) string {
	store[key] = value

	return ""
}

func get(key string) interface{} {
	return store[key]
}

// Parse creates a template instance from string template
func Parse(ID string, template string) (*tmpl.Template, error) {
	funcMap := sprig.FuncMap()
	funcMap["isset"] = isset
	funcMap["set"] = set
	funcMap["get"] = get

	return tmpl.New(ID).Funcs(funcMap).Parse(template)
}

// Build creates a template instance and runs it against datas to get
// final resolved string
func Build(ID string, template string, data interface{}) (string, error) {
	t, err := Parse(ID, template)

	if err != nil {
		return "", errh.AddCustomMessageToError("check your template is well-formed", err)
	}

	b := bytes.Buffer{}

	if err = t.Execute(&b, data); err != nil {
		return "", err
	}

	return b.String(), nil
}
