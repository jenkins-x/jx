package config

import (
	"github.com/antham/envh"
)

type envDecoratorConfigurator struct {
	config *envh.EnvTree
}

func (e *envDecoratorConfigurator) process(config *CHYLE) (bool, error) {
	if e.isDisabled() {
		return true, nil
	}

	config.FEATURES.DECORATORS.ENABLED = true
	config.FEATURES.DECORATORS.ENV = true

	for _, f := range []func() error{
		e.validateEnvironmentVariables,
	} {
		if err := f(); err != nil {
			return true, err
		}
	}

	e.setEnvConfigs(config)

	return true, nil
}

func (e *envDecoratorConfigurator) isDisabled() bool {
	return featureDisabled(e.config, [][]string{{"CHYLE", "DECORATORS", "ENV"}})
}

func (e *envDecoratorConfigurator) validateEnvironmentVariables() error {
	for _, key := range e.config.FindChildrenKeysUnsecured("CHYLE", "DECORATORS", "ENV") {
		if err := validateEnvironmentVariablesDefinition(e.config, [][]string{{"CHYLE", "DECORATORS", "ENV", key, "DESTKEY"}, {"CHYLE", "DECORATORS", "ENV", key, "VARNAME"}}); err != nil {
			return err
		}
	}

	return nil
}

func (e *envDecoratorConfigurator) setEnvConfigs(config *CHYLE) {
	config.DECORATORS.ENV = map[string]struct {
		DESTKEY string
		VARNAME string
	}{}

	for _, key := range e.config.FindChildrenKeysUnsecured("CHYLE", "DECORATORS", "ENV") {
		config.DECORATORS.ENV[key] = struct {
			DESTKEY string
			VARNAME string
		}{
			e.config.FindStringUnsecured("CHYLE", "DECORATORS", "ENV", key, "DESTKEY"),
			e.config.FindStringUnsecured("CHYLE", "DECORATORS", "ENV", key, "VARNAME"),
		}
	}
}
