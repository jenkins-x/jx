package config

import (
	"github.com/antham/envh"
)

type customAPISenderConfigurator struct {
	config *envh.EnvTree
}

func (g *customAPISenderConfigurator) process(config *CHYLE) (bool, error) {
	if g.isDisabled() {
		return false, nil
	}

	config.FEATURES.SENDERS.ENABLED = true
	config.FEATURES.SENDERS.CUSTOMAPI = true

	for _, f := range []func() error{
		g.validateCredentials,
		g.validateMandatoryFields,
		g.validateURL,
	} {
		if err := f(); err != nil {
			return false, err
		}
	}

	return false, nil
}

func (g *customAPISenderConfigurator) isDisabled() bool {
	return !g.config.IsExistingSubTree("CHYLE", "SENDERS", "CUSTOMAPI")
}

func (g *customAPISenderConfigurator) validateCredentials() error {
	return validateEnvironmentVariablesDefinition(g.config, [][]string{{"CHYLE", "SENDERS", "CUSTOMAPI", "CREDENTIALS", "TOKEN"}})
}

func (g *customAPISenderConfigurator) validateMandatoryFields() error {
	return validateEnvironmentVariablesDefinition(g.config, [][]string{{"CHYLE", "SENDERS", "CUSTOMAPI", "ENDPOINT", "URL"}})
}

func (g *customAPISenderConfigurator) validateURL() error {
	return validateURL(g.config, []string{"CHYLE", "SENDERS", "CUSTOMAPI", "ENDPOINT", "URL"})
}
