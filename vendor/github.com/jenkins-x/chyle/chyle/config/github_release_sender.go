package config

import (
	"github.com/antham/envh"
)

type githubReleaseSenderConfigurator struct {
	config *envh.EnvTree
}

func (g *githubReleaseSenderConfigurator) process(config *CHYLE) (bool, error) {
	if g.isDisabled() {
		return false, nil
	}

	config.FEATURES.SENDERS.ENABLED = true
	config.FEATURES.SENDERS.GITHUBRELEASE = true

	for _, f := range []func() error{
		g.validateCredentials,
		g.validateReleaseMandatoryFields,
		g.validateRepositoryName,
	} {
		if err := f(); err != nil {
			return false, err
		}
	}

	return false, nil
}

func (g *githubReleaseSenderConfigurator) isDisabled() bool {
	return !g.config.IsExistingSubTree("CHYLE", "SENDERS", "GITHUBRELEASE")
}

func (g *githubReleaseSenderConfigurator) validateCredentials() error {
	return validateEnvironmentVariablesDefinition(g.config, [][]string{{"CHYLE", "SENDERS", "GITHUBRELEASE", "CREDENTIALS", "OAUTHTOKEN"}, {"CHYLE", "SENDERS", "GITHUBRELEASE", "CREDENTIALS", "OWNER"}})
}

func (g *githubReleaseSenderConfigurator) validateReleaseMandatoryFields() error {
	if err := validateEnvironmentVariablesDefinition(g.config, [][]string{{"CHYLE", "SENDERS", "GITHUBRELEASE", "RELEASE", "TAGNAME"}, {"CHYLE", "SENDERS", "GITHUBRELEASE", "RELEASE", "TEMPLATE"}}); err != nil {
		return err
	}

	if err := validateTemplate(g.config, []string{"CHYLE", "SENDERS", "GITHUBRELEASE", "RELEASE", "TEMPLATE"}); err != nil {
		return err
	}

	return nil
}

func (g *githubReleaseSenderConfigurator) validateRepositoryName() error {
	return validateEnvironmentVariablesDefinition(g.config, [][]string{{"CHYLE", "SENDERS", "GITHUBRELEASE", "REPOSITORY", "NAME"}})
}
