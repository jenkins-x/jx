package config

import (
	"fmt"
	"strings"

	"github.com/antham/envh"
)

type stdoutSenderConfigurator struct {
	config *envh.EnvTree
}

func (s *stdoutSenderConfigurator) process(config *CHYLE) (bool, error) {
	if s.isDisabled() {
		return false, nil
	}

	config.FEATURES.SENDERS.ENABLED = true
	config.FEATURES.SENDERS.STDOUT = true

	return false, s.validateFormat()
}

func (s *stdoutSenderConfigurator) isDisabled() bool {
	return featureDisabled(s.config, [][]string{{"CHYLE", "SENDERS", "STDOUT"}})
}

func (s *stdoutSenderConfigurator) validateFormat() error {
	var err error
	var format string
	keyChain := []string{"CHYLE", "SENDERS", "STDOUT"}

	if format, err = s.config.FindString(append(keyChain, "FORMAT")...); err != nil {
		return MissingEnvError{[]string{strings.Join(append(keyChain, "FORMAT"), "_")}}
	}

	switch format {
	case "json":
		return nil
	case "template":
		return s.validateTemplateFormat()
	}

	return EnvValidationError{fmt.Sprintf(`"CHYLE_SENDERS_STDOUT_FORMAT" "%s" doesn't exist`, format), "CHYLE_SENDERS_STDOUT_FORMAT"}
}

func (s *stdoutSenderConfigurator) validateTemplateFormat() error {
	tmplKeyChain := []string{"CHYLE", "SENDERS", "STDOUT", "TEMPLATE"}

	if ok, err := s.config.HasSubTreeValue(tmplKeyChain...); !ok || err != nil {
		return MissingEnvError{[]string{strings.Join(tmplKeyChain, "_")}}
	}

	if err := validateTemplate(s.config, tmplKeyChain); err != nil {
		return err
	}

	return nil
}
