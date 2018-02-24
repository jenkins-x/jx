package prompt

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"

	"github.com/antham/chyle/chyle/tmplh"
)

func mergePrompters(prompters ...[]strumt.Prompter) []strumt.Prompter {
	results := prompters[0]

	for _, p := range prompters[1:] {
		results = append(results, p...)
	}

	return results
}

func addMainMenuChoice(choices []builder.SwitchConfig) []builder.SwitchConfig {
	return append(
		choices,
		builder.SwitchConfig{
			Choice:       "m",
			PromptString: "Menu",
			NextPromptID: "mainMenu",
		},
	)
}

func addQuitChoice(choices []builder.SwitchConfig) []builder.SwitchConfig {
	return append(
		choices,
		builder.SwitchConfig{
			Choice:       "q",
			PromptString: "Dump generated configuration and quit",
			NextPromptID: "",
		})
}

func addMainMenuAndQuitChoice(choices []builder.SwitchConfig) []builder.SwitchConfig {
	return addQuitChoice(addMainMenuChoice(choices))
}

func noOpValidator(value string) error {
	return nil
}

func validateDefinedValue(value string) error {
	if value == "" {
		return fmt.Errorf("No value given")
	}

	return nil
}

func validateRegexp(value string) error {
	if _, err := regexp.Compile(value); err != nil {
		return fmt.Errorf(`"%s" is an invalid regexp : %s`, value, err.Error())
	}

	return nil
}

func validateURL(value string) error {
	if _, err := url.ParseRequestURI(value); err != nil {
		return fmt.Errorf(`"%s" must be a valid URL`, value)
	}

	return nil
}

func validateBoolean(value string) error {
	if _, err := strconv.ParseBool(value); err != nil {
		return fmt.Errorf(`"%s" must be true or false`, value)
	}

	return nil
}

func noOpRunBeforeNextPrompt(value string, store *builder.Store) {}

func validateTemplate(value string) error {
	if _, err := tmplh.Parse("template", value); err != nil {
		return fmt.Errorf(`"%s" is an invalid template : %s`, value, err.Error())
	}

	return nil
}
