package util

import (
	"fmt"
	"sort"

	"gopkg.in/AlecAivazis/survey.v1"
)

func PickValue(message string, defaultValue string, required bool) (string, error) {
	answer := ""
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
	}
	validator := survey.Required
	if !required {
		validator = nil
	}
	err := survey.AskOne(prompt, &answer, validator)
	if err != nil {
		return "", err
	}
	return answer, nil
}

func PickNameWithDefault(names []string, message string, defaultValue string) (string, error) {
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
		err := survey.AskOne(prompt, &name, nil)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}

func PickRequiredNameWithDefault(names []string, message string, defaultValue string) (string, error) {
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
		err := survey.AskOne(prompt, &name, survey.Required)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}

func PickName(names []string, message string) (string, error) {
	return PickNameWithDefault(names, message, "")
}

func PickNames(names []string, message string) ([]string, error) {
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
		err := survey.AskOne(prompt, &picked, nil)
		if err != nil {
			return picked, err
		}
	}
	return picked, nil
}

// SelectNames select which names from the list should be chosen
func SelectNames(names []string, message string, selectAll bool) ([]string, error) {
	answer := []string{}
	if len(names) == 0 {
		return answer, fmt.Errorf("No names to choose from!")
	}
	sort.Strings(names)

	prompt := &survey.MultiSelect{
		Message: message,
		Options: names,
	}
	if selectAll {
		prompt.Default = names
	}
	err := survey.AskOne(prompt, &answer, nil)
	return answer, err
}

// Confirm prompts the user to confirm something
func Confirm(message string, defaultValue bool, help string) bool {
	answer := defaultValue
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
		Help:    help,
	}
	survey.AskOne(prompt, &answer, nil)
	return answer
}
