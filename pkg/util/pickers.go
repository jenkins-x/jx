package util

import "gopkg.in/AlecAivazis/survey.v1"

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
