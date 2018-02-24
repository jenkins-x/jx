package config

import (
	"github.com/antham/envh"
)

type shellDecoratorConfigurator struct {
	config *envh.EnvTree
}

func (s *shellDecoratorConfigurator) process(config *CHYLE) (bool, error) {
	if s.isDisabled() {
		return true, nil
	}

	config.FEATURES.DECORATORS.ENABLED = true
	config.FEATURES.DECORATORS.SHELL = true

	for _, f := range []func() error{
		s.validateShellConfigs,
	} {
		if err := f(); err != nil {
			return true, err
		}
	}

	s.setShellConfigs(config)

	return true, nil
}

func (s *shellDecoratorConfigurator) isDisabled() bool {
	return featureDisabled(s.config, [][]string{
		{"CHYLE", "DECORATORS", "SHELL"},
	})
}

func (s *shellDecoratorConfigurator) validateShellConfigs() error {
	for _, key := range s.config.FindChildrenKeysUnsecured("CHYLE", "DECORATORS", "SHELL") {
		if err := validateEnvironmentVariablesDefinition(s.config, [][]string{{"CHYLE", "DECORATORS", "SHELL", key, "DESTKEY"}, {"CHYLE", "DECORATORS", "SHELL", key, "ORIGKEY"}, {"CHYLE", "DECORATORS", "SHELL", key, "COMMAND"}}); err != nil {
			return err
		}
	}

	return nil
}

func (s *shellDecoratorConfigurator) setShellConfigs(config *CHYLE) {
	config.DECORATORS.SHELL = map[string]struct {
		COMMAND string
		ORIGKEY string
		DESTKEY string
	}{}

	for _, key := range s.config.FindChildrenKeysUnsecured("CHYLE", "DECORATORS", "SHELL") {
		config.DECORATORS.SHELL[key] = struct {
			COMMAND string
			ORIGKEY string
			DESTKEY string
		}{
			s.config.FindStringUnsecured("CHYLE", "DECORATORS", "SHELL", key, "COMMAND"),
			s.config.FindStringUnsecured("CHYLE", "DECORATORS", "SHELL", key, "ORIGKEY"),
			s.config.FindStringUnsecured("CHYLE", "DECORATORS", "SHELL", key, "DESTKEY"),
		}
	}

}
