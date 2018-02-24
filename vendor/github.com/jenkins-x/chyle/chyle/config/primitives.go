package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/antham/chyle/chyle/tmplh"

	"github.com/antham/envh"
)

// MissingEnvError is called when one or several
// environment variables are missing
type MissingEnvError struct {
	envs []string
}

// Envs returns environment variables missing
func (e MissingEnvError) Envs() []string {
	return e.envs
}

// Errors returns error as string
func (e MissingEnvError) Error() string {
	switch len(e.envs) {
	case 1:
		return fmt.Sprintf(`environment variable missing : "%s"`, e.envs[0])
	default:
		return fmt.Sprintf(`environments variables missing : "%s"`, strings.Join(e.envs, `", "`))
	}
}

func validateEnvironmentVariablesDefinition(conf *envh.EnvTree, keyChains [][]string) error {
	undefinedKeys := []string{}

	for _, keyChain := range keyChains {
		ok, err := conf.HasSubTreeValue(keyChain...)

		if !ok || err != nil {
			undefinedKeys = append(undefinedKeys, strings.Join(keyChain, "_"))
		}
	}

	if len(undefinedKeys) > 0 {
		return MissingEnvError{undefinedKeys}
	}

	return nil
}

func validateStringValue(value string, conf *envh.EnvTree, keyChain []string) error {
	if conf.FindStringUnsecured(keyChain...) != value {
		return EnvValidationError{fmt.Sprintf(`variable %s must be equal to "%s"`, strings.Join(keyChain, "_"), value), strings.Join(keyChain, "_")}
	}

	return nil
}

func validateURL(fullconfig *envh.EnvTree, chain []string) error {
	if _, err := url.ParseRequestURI(fullconfig.FindStringUnsecured(chain...)); err != nil {
		return EnvValidationError{fmt.Sprintf(`provide a valid URL for "%s", "%s" given`, strings.Join(chain, "_"), fullconfig.FindStringUnsecured(chain...)), strings.Join(chain, "_")}
	}

	return nil
}

func validateRegexp(fullconfig *envh.EnvTree, keyChain []string) error {
	if _, err := regexp.Compile(fullconfig.FindStringUnsecured(keyChain...)); err != nil {
		return EnvValidationError{fmt.Sprintf(`provide a valid regexp for "%s", "%s" given`, strings.Join(keyChain, "_"), fullconfig.FindStringUnsecured(keyChain...)), strings.Join(keyChain, "_")}
	}

	return nil
}

func validateOneOf(fullconfig *envh.EnvTree, keyChain []string, choices []string) error {
	val := fullconfig.FindStringUnsecured(keyChain...)

	for _, choice := range choices {
		if choice == val {
			return nil
		}
	}

	return EnvValidationError{fmt.Sprintf(`provide a value for "%s" from one of those values : ["%s"], "%s" given`, strings.Join(keyChain, "_"), strings.Join(choices, `", "`), val), strings.Join(keyChain, "_")}
}

func validateTemplate(fullconfig *envh.EnvTree, keyChain []string) error {
	val := fullconfig.FindStringUnsecured(keyChain...)

	_, err := tmplh.Parse("test", val)

	if err != nil {
		return EnvValidationError{fmt.Sprintf(`provide a valid template string for "%s" : "%s", "%s" given`, strings.Join(keyChain, "_"), err.Error(), val), strings.Join(keyChain, "_")}
	}

	return nil
}

// featureDisabled return false if one subtree declared in keyChains exists
func featureDisabled(fullconfig *envh.EnvTree, keyChains [][]string) bool {
	for _, keyChain := range keyChains {
		if fullconfig.IsExistingSubTree(keyChain...) {
			return false
		}
	}

	return true
}
