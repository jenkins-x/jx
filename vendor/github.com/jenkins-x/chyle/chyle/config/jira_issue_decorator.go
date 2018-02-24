package config

import (
	"github.com/antham/envh"
)

func getJiraIssueDecoratorMandatoryParamsRefs() []ref {
	return []ref{
		{
			&chyleConfig.DECORATORS.JIRAISSUE.ENDPOINT.URL,
			[]string{"CHYLE", "DECORATORS", "JIRAISSUE", "ENDPOINT", "URL"},
		},
		{
			&chyleConfig.DECORATORS.JIRAISSUE.CREDENTIALS.USERNAME,
			[]string{"CHYLE", "DECORATORS", "JIRAISSUE", "CREDENTIALS", "USERNAME"},
		},
		{
			&chyleConfig.DECORATORS.JIRAISSUE.CREDENTIALS.PASSWORD,
			[]string{"CHYLE", "DECORATORS", "JIRAISSUE", "CREDENTIALS", "PASSWORD"},
		},
	}
}

func getJiraIssueDecoratorFeatureRefs() []*bool {
	return []*bool{
		&chyleConfig.FEATURES.DECORATORS.ENABLED,
		&chyleConfig.FEATURES.DECORATORS.JIRAISSUE,
	}
}

func getJiraIssueDecoratorCustomValidationFuncs(config *envh.EnvTree) []func() error {
	return []func() error{}
}

func getJiraIssueDecoratorCustomSettersFuncs() []func(*CHYLE) {
	return []func(*CHYLE){}
}

func newJiraIssueDecoratorConfigurator(config *envh.EnvTree) configurater {
	return &apiDecoratorConfigurator{
		config: config,
		apiDecoratorConfig: apiDecoratorConfig{
			"JIRAISSUEID",
			"jiraIssueId",
			"JIRAISSUE",
			&chyleConfig.DECORATORS.JIRAISSUE.KEYS,
			getJiraIssueDecoratorMandatoryParamsRefs(),
			getJiraIssueDecoratorFeatureRefs(),
			getJiraIssueDecoratorCustomValidationFuncs(config),
			getJiraIssueDecoratorCustomSettersFuncs(),
		},
	}
}
