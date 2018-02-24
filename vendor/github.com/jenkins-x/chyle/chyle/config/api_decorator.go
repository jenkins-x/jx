package config

import (
	"fmt"
	"strings"

	"github.com/antham/envh"
)

// customAPIValidators defines validators called when last key of a key chain matches
// a key defined in map
var customAPIValidators = map[string]func(*envh.EnvTree, []string) error{
	"URL": validateURL,
}

// codebeat:disable[TOO_MANY_IVARS]

type apiDecoratorConfig struct {
	extractorKey          string
	extractorDestKeyValue string
	decoratorKey          string
	keysRef               *map[string]struct {
		DESTKEY string
		FIELD   string
	}
	mandatoryParamsRefs   []ref
	featureRefs           []*bool
	customValidationFuncs []func() error
	customSetterFuncs     []func(*CHYLE)
}

// codebeat:enable[TOO_MANY_IVARS]

// apiDecoratorConfigurator is a generic api
// decorator configurator
type apiDecoratorConfigurator struct {
	config *envh.EnvTree
	apiDecoratorConfig
}

func (a *apiDecoratorConfigurator) process(config *CHYLE) (bool, error) {
	if a.isDisabled() {
		return true, nil
	}

	for _, featureRef := range a.featureRefs {
		*featureRef = true
	}

	if err := a.validate(); err != nil {
		return true, err
	}

	a.set(config)

	return true, nil
}

func (a *apiDecoratorConfigurator) isDisabled() bool {
	return featureDisabled(a.config, [][]string{
		{"CHYLE", "DECORATORS", a.decoratorKey},
		{"CHYLE", "EXTRACTORS", a.extractorKey},
	})
}

func (a *apiDecoratorConfigurator) validate() error {
	for _, f := range append([]func() error{
		a.validateMandatoryParameters,
		a.validateKeys,
		a.validateExtractor,
	}, a.customValidationFuncs...) {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (a *apiDecoratorConfigurator) set(config *CHYLE) {
	for _, f := range append([]func(*CHYLE){
		a.setKeys,
		a.setMandatoryParameters,
	}, a.customSetterFuncs...) {
		f(config)
	}
}

// validateExtractor checks if an extractor is defined to get
// data needed to contact remote api
func (a *apiDecoratorConfigurator) validateExtractor() error {
	if err := validateEnvironmentVariablesDefinition(
		a.config,
		[][]string{
			{"CHYLE", "EXTRACTORS", a.extractorKey, "ORIGKEY"},
			{"CHYLE", "EXTRACTORS", a.extractorKey, "DESTKEY"},
			{"CHYLE", "EXTRACTORS", a.extractorKey, "REG"},
		},
	); err != nil {
		return err
	}

	if err := validateStringValue(a.extractorDestKeyValue, a.config, []string{"CHYLE", "EXTRACTORS", a.extractorKey, "DESTKEY"}); err != nil {
		return err
	}

	return nil
}

func (a *apiDecoratorConfigurator) validateMandatoryParameters() error {
	keyChains := [][]string{}

	for _, ref := range a.mandatoryParamsRefs {
		keyChains = append(keyChains, ref.keyChain)
	}

	if err := validateEnvironmentVariablesDefinition(a.config, keyChains); err != nil {
		return err
	}

	return a.applyCustomValidators(&keyChains)
}

// applyCustomValidators applies validators defined in map customAPIValidators
func (a *apiDecoratorConfigurator) applyCustomValidators(keyChains *[][]string) error {
	for _, keyChain := range *keyChains {
		f, ok := customAPIValidators[keyChain[len(keyChain)-1]]

		if !ok {
			continue
		}

		if err := f(a.config, keyChain); err != nil {
			return err
		}
	}

	return nil
}

// validateKeys checks key mapping between fields extracted from api and fields added to final struct
func (a *apiDecoratorConfigurator) validateKeys() error {
	keys, err := a.config.FindChildrenKeys("CHYLE", "DECORATORS", a.decoratorKey, "KEYS")

	if err != nil {
		return EnvValidationError{fmt.Sprintf(`define at least one environment variable couple "CHYLE_DECORATORS_%s_KEYS_*_DESTKEY" and "CHYLE_DECORATORS_%s_KEYS_*_FIELD", replace "*" with your own naming`, a.decoratorKey, a.decoratorKey), strings.Join([]string{"CHYLE", "DECORATORS", a.decoratorKey, "KEYS"}, "_")}
	}

	for _, key := range keys {
		if err := validateEnvironmentVariablesDefinition(a.config, [][]string{{"CHYLE", "DECORATORS", a.decoratorKey, "KEYS", key, "DESTKEY"}, {"CHYLE", "DECORATORS", a.decoratorKey, "KEYS", key, "FIELD"}}); err != nil {
			return err
		}
	}

	return nil
}

func (a *apiDecoratorConfigurator) setMandatoryParameters(config *CHYLE) {
	for _, c := range a.mandatoryParamsRefs {
		*(c.ref) = a.config.FindStringUnsecured(c.keyChain...)
	}
}

func (a *apiDecoratorConfigurator) setKeys(config *CHYLE) {
	ref := a.keysRef
	*ref = map[string]struct {
		DESTKEY string
		FIELD   string
	}{}

	for _, key := range a.config.FindChildrenKeysUnsecured("CHYLE", "DECORATORS", a.decoratorKey, "KEYS") {
		(*ref)[key] = struct {
			DESTKEY string
			FIELD   string
		}{
			a.config.FindStringUnsecured("CHYLE", "DECORATORS", a.decoratorKey, "KEYS", key, "DESTKEY"),
			a.config.FindStringUnsecured("CHYLE", "DECORATORS", a.decoratorKey, "KEYS", key, "FIELD"),
		}
	}
}
