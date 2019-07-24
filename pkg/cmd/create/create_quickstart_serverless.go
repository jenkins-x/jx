package create

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// ExecCommand allows easy fakes of `exec.command``
var ExecCommand = exec.Command

// GetServerlessTemplates returns the templates supported by `serverless` CLI`
var GetServerlessTemplates = func() ([]string, error) {
	cmd := ExecCommand("serverless", "create", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}, err
	}
	lines := strings.Split(string(out), "\n")
	templateLine := ""
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--template") {
			templateLine = line
			break
		}
	}
	if len(templateLine) == 0 {
		return []string{}, errors.New("The output does not contain the line starting with `--template`\n" + string(out))
	}
	templates := []string{}
	for _, template := range strings.Split(templateLine, ",") {
		if strings.Contains(template, " and ") {
			templates = append(templates, strings.Split(template, " and ")[0], strings.Split(template, " and ")[1])
		} else {
			templates = append(templates, template)
		}
	}
	formattedTemplates := []string{}
	for _, template := range templates {
		template = strings.ReplaceAll(template, "\"", "")
		template = strings.TrimSpace(template)
		if strings.HasPrefix(template, "--template") {
			searchString := "Available templates:"
			index := strings.Index(template, searchString)
			if index == -1 {
				return []string{}, errors.New("Could not find `Available templates`")
			}
			template = template[index+len(searchString)+1 : len(template)]
		}
		formattedTemplates = append(formattedTemplates, template)
	}
	return formattedTemplates, nil
}
