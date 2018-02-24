package config

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/antham/envh"

	"github.com/antham/chyle/chyle/decorators"
	"github.com/antham/chyle/chyle/extractors"
	"github.com/antham/chyle/chyle/matchers"
	"github.com/antham/chyle/chyle/senders"
)

var chyleConfig CHYLE

// EnvValidationError is called when validating a
// configuration failed, it keeps a track of which
// environment variable is actually tested
type EnvValidationError struct {
	message string
	env     string
}

// Env returns environment variable currently tested
func (v EnvValidationError) Env() string {
	return v.env
}

// Error returns error as string
func (v EnvValidationError) Error() string {
	return v.message
}

// configurater must be implemented to process custom config
type configurater interface {
	process(config *CHYLE) (bool, error)
}

type ref struct {
	ref      *string
	keyChain []string
}

// codebeat:disable[TOO_MANY_IVARS]

// CHYLE hold config extracted from environment variables
type CHYLE struct {
	FEATURES struct {
		MATCHERS   matchers.Features
		EXTRACTORS extractors.Features
		DECORATORS decorators.Features
		SENDERS    senders.Features
	} `json:"-"`
	GIT struct {
		REPOSITORY struct {
			PATH string
		}
		REFERENCE struct {
			FROM string
			TO   string
		}
	}
	MATCHERS   matchers.Config
	EXTRACTORS extractors.Config
	DECORATORS decorators.Config
	SENDERS    senders.Config
}

// codebeat:enable[TOO_MANY_IVARS]

// Walk traverses struct to populate or validate fields
func (c *CHYLE) Walk(fullconfig *envh.EnvTree, keyChain []string) (bool, error) {
	if walker, ok := map[string]func(*envh.EnvTree, []string) (bool, error){
		"CHYLE_FEATURES":       func(*envh.EnvTree, []string) (bool, error) { return true, nil },
		"CHYLE_GIT_REFERENCE":  c.validateChyleGitReference,
		"CHYLE_GIT_REPOSITORY": c.validateChyleGitRepository,
	}[strings.Join(keyChain, "_")]; ok {
		return walker(fullconfig, keyChain)
	}

	if processor, ok := map[string]func() configurater{
		"CHYLE_DECORATORS_ENV":         func() configurater { return &envDecoratorConfigurator{fullconfig} },
		"CHYLE_DECORATORS_CUSTOMAPI":   func() configurater { return newCustomAPIDecoratorConfigurator(fullconfig) },
		"CHYLE_DECORATORS_GITHUBISSUE": func() configurater { return newGithubIssueDecoratorConfigurator(fullconfig) },
		"CHYLE_DECORATORS_JIRAISSUE":   func() configurater { return newJiraIssueDecoratorConfigurator(fullconfig) },
		"CHYLE_DECORATORS_SHELL":       func() configurater { return &shellDecoratorConfigurator{fullconfig} },
		"CHYLE_EXTRACTORS":             func() configurater { return &extractorsConfigurator{fullconfig} },
		"CHYLE_MATCHERS":               func() configurater { return &matchersConfigurator{fullconfig} },
		"CHYLE_SENDERS_GITHUBRELEASE":  func() configurater { return &githubReleaseSenderConfigurator{fullconfig} },
		"CHYLE_SENDERS_CUSTOMAPI":      func() configurater { return &customAPISenderConfigurator{fullconfig} },
		"CHYLE_SENDERS_STDOUT":         func() configurater { return &stdoutSenderConfigurator{fullconfig} },
	}[strings.Join(keyChain, "_")]; ok {
		return processor().process(c)
	}

	return false, nil
}

func (c *CHYLE) validateChyleGitRepository(fullconfig *envh.EnvTree, keyChain []string) (bool, error) {
	return false, validateEnvironmentVariablesDefinition(fullconfig, [][]string{{"CHYLE", "GIT", "REPOSITORY", "PATH"}})
}

func (c *CHYLE) validateChyleGitReference(fullconfig *envh.EnvTree, keyChain []string) (bool, error) {
	return false, validateEnvironmentVariablesDefinition(fullconfig, [][]string{{"CHYLE", "GIT", "REFERENCE", "FROM"}, {"CHYLE", "GIT", "REFERENCE", "TO"}})
}

// Create returns app config from an EnvTree object
func Create(envConfig *envh.EnvTree) (*CHYLE, error) {
	chyleConfig = CHYLE{}
	return &chyleConfig, envConfig.PopulateStruct(&chyleConfig)
}

// Debug dumps given CHYLE config as JSON structure
func Debug(config *CHYLE, logger *log.Logger) {
	if d, err := json.MarshalIndent(config, "", "    "); err == nil {
		logger.Println(string(d))
	}
}
