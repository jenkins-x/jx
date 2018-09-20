package util

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func PickValue(message string, defaultValue string, required bool, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
	answer := ""
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
	}
	validator := survey.Required
	if !required {
		validator = nil
	}
	surveyOpts := survey.WithStdio(in, out, outErr)
	err := survey.AskOne(prompt, &answer, validator, surveyOpts)
	if err != nil {
		return "", err
	}
	return answer, nil
}

func PickPassword(message string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
	answer := ""
	prompt := &survey.Password{
		Message: message,
	}
	validator := survey.Required
	surveyOpts := survey.WithStdio(in, out, outErr)
	err := survey.AskOne(prompt, &answer, validator, surveyOpts)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

func PickNameWithDefault(names []string, message string, defaultValue string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
	name := ""
	if len(names) == 0 {
		return "", nil
	} else if len(names) == 1 {
		name = names[0]
	} else {
		prompt := &survey.Select{
			Message: message,
			Options: names,
			Default: defaultValue,
		}
		surveyOpts := survey.WithStdio(in, out, outErr)
		err := survey.AskOne(prompt, &name, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}

func PickRequiredNameWithDefault(names []string, message string, defaultValue string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
	name := ""
	if len(names) == 0 {
		return "", nil
	} else if len(names) == 1 {
		name = names[0]
	} else {
		prompt := &survey.Select{
			Message: message,
			Options: names,
			Default: defaultValue,
		}
		surveyOpts := survey.WithStdio(in, out, outErr)
		err := survey.AskOne(prompt, &name, survey.Required, surveyOpts)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}

func PickName(names []string, message string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
	return PickNameWithDefault(names, message, "", in, out, outErr)
}

func PickNames(names []string, message string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) ([]string, error) {
	picked := []string{}
	if len(names) == 0 {
		return picked, nil
	} else if len(names) == 1 {
		return names, nil
	} else {
		prompt := &survey.MultiSelect{
			Message: message,
			Options: names,
		}
		surveyOpts := survey.WithStdio(in, out, outErr)
		err := survey.AskOne(prompt, &picked, nil, surveyOpts)
		if err != nil {
			return picked, err
		}
	}
	return picked, nil
}

// SelectNamesWithFilter selects from a list of names with a given filter. Optionally selecting them all
func SelectNamesWithFilter(names []string, message string, selectAll bool, filter string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) ([]string, error) {
	filtered := []string{}
	for _, name := range names {
		if filter == "" || strings.Index(name, filter) >= 0 {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("No names match filter: %s", filter)
	}
	return SelectNames(filtered, message, selectAll, in, out, outErr)
}

// SelectNames select which names from the list should be chosen
func SelectNames(names []string, message string, selectAll bool, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) ([]string, error) {
	answer := []string{}
	if len(names) == 0 {
		return answer, fmt.Errorf("No names to choose from")
	}
	sort.Strings(names)

	prompt := &survey.MultiSelect{
		Message: message,
		Options: names,
	}
	if selectAll {
		prompt.Default = names
	}
	surveyOpts := survey.WithStdio(in, out, outErr)
	err := survey.AskOne(prompt, &answer, nil, surveyOpts)
	return answer, err
}

// Confirm prompts the user to confirm something
func Confirm(message string, defaultValue bool, help string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) bool {
	answer := defaultValue
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
		Help:    help,
	}
	surveyOpts := survey.WithStdio(in, out, outErr)
	survey.AskOne(prompt, &answer, nil, surveyOpts)
	log.Blank()
	return answer
}
