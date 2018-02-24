package config

import (
	"regexp"

	"github.com/antham/envh"
)

type extractorsConfigurator struct {
	config *envh.EnvTree
}

func (e *extractorsConfigurator) process(config *CHYLE) (bool, error) {
	if e.isDisabled() {
		return true, nil
	}

	config.FEATURES.EXTRACTORS.ENABLED = true

	for _, f := range []func() error{
		e.validateExtractors,
	} {
		if err := f(); err != nil {
			return true, err
		}
	}

	e.setExtractors(config)

	return true, nil
}

func (e *extractorsConfigurator) isDisabled() bool {
	return featureDisabled(e.config, [][]string{{"CHYLE", "EXTRACTORS"}})
}

func (e *extractorsConfigurator) validateExtractors() error {
	for _, key := range e.config.FindChildrenKeysUnsecured("CHYLE", "EXTRACTORS") {
		if err := validateEnvironmentVariablesDefinition(e.config, [][]string{{"CHYLE", "EXTRACTORS", key, "ORIGKEY"}, {"CHYLE", "EXTRACTORS", key, "DESTKEY"}, {"CHYLE", "EXTRACTORS", key, "REG"}}); err != nil {
			return err
		}

		if err := validateRegexp(e.config, []string{"CHYLE", "EXTRACTORS", key, "REG"}); err != nil {
			return err
		}
	}

	return nil
}

func (e *extractorsConfigurator) setExtractors(config *CHYLE) {
	config.EXTRACTORS = map[string]struct {
		ORIGKEY string
		DESTKEY string
		REG     *regexp.Regexp
	}{}

	for _, key := range e.config.FindChildrenKeysUnsecured("CHYLE", "EXTRACTORS") {
		config.EXTRACTORS[key] = struct {
			ORIGKEY string
			DESTKEY string
			REG     *regexp.Regexp
		}{

			e.config.FindStringUnsecured("CHYLE", "EXTRACTORS", key, "ORIGKEY"),
			e.config.FindStringUnsecured("CHYLE", "EXTRACTORS", key, "DESTKEY"),
			regexp.MustCompile(e.config.FindStringUnsecured("CHYLE", "EXTRACTORS", key, "REG")),
		}
	}
}
