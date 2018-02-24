package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antham/envh"
)

func getCustomAPIDecoratorMandatoryParamsRefs() []ref {
	return []ref{
		{
			&chyleConfig.DECORATORS.CUSTOMAPI.ENDPOINT.URL,
			[]string{"CHYLE", "DECORATORS", "CUSTOMAPI", "ENDPOINT", "URL"},
		},
		{
			&chyleConfig.DECORATORS.CUSTOMAPI.CREDENTIALS.TOKEN,
			[]string{"CHYLE", "DECORATORS", "CUSTOMAPI", "CREDENTIALS", "TOKEN"},
		},
	}
}

func getCustomAPIDecoratorFeatureRefs() []*bool {
	return []*bool{
		&chyleConfig.FEATURES.DECORATORS.ENABLED,
		&chyleConfig.FEATURES.DECORATORS.CUSTOMAPI,
	}
}

func getCustomAPIDecoratorCustomValidationFuncs(config *envh.EnvTree) []func() error {
	return []func() error{
		func() error {
			keyChain := []string{"CHYLE", "DECORATORS", "CUSTOMAPI", "ENDPOINT", "URL"}
			URL := config.FindStringUnsecured(keyChain...)

			if !regexp.MustCompile(`{{\s*ID\s*}}`).MatchString(URL) {
				return EnvValidationError{fmt.Sprintf(`ensure you defined a placeholder {{ID}} in URL defined in "%s"`, strings.Join(keyChain, "_")), strings.Join(keyChain, "_")}
			}

			return nil
		},
	}
}

func getCustomAPIDecoratorCustomSettersFuncs() []func(*CHYLE) {
	return []func(*CHYLE){}
}

func newCustomAPIDecoratorConfigurator(config *envh.EnvTree) configurater {
	return &apiDecoratorConfigurator{
		config: config,
		apiDecoratorConfig: apiDecoratorConfig{
			"CUSTOMAPIID",
			"customApiId",
			"CUSTOMAPI",
			&chyleConfig.DECORATORS.CUSTOMAPI.KEYS,
			getCustomAPIDecoratorMandatoryParamsRefs(),
			getCustomAPIDecoratorFeatureRefs(),
			getCustomAPIDecoratorCustomValidationFuncs(config),
			getCustomAPIDecoratorCustomSettersFuncs(),
		},
	}
}
