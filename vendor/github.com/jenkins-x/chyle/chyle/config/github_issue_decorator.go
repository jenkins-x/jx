package config

import (
	"github.com/antham/envh"
)

func getGithubIssueDecoratorMandatoryParamsRefs() []ref {
	return []ref{
		{
			&chyleConfig.DECORATORS.GITHUBISSUE.CREDENTIALS.OAUTHTOKEN,
			[]string{"CHYLE", "DECORATORS", "GITHUBISSUE", "CREDENTIALS", "OAUTHTOKEN"},
		},
		{
			&chyleConfig.DECORATORS.GITHUBISSUE.CREDENTIALS.OWNER,
			[]string{"CHYLE", "DECORATORS", "GITHUBISSUE", "CREDENTIALS", "OWNER"},
		},
		{
			&chyleConfig.DECORATORS.GITHUBISSUE.REPOSITORY.NAME,
			[]string{"CHYLE", "DECORATORS", "GITHUBISSUE", "REPOSITORY", "NAME"},
		},
	}
}

func getGithubIssueDecoratorFeatureRefs() []*bool {
	return []*bool{
		&chyleConfig.FEATURES.DECORATORS.ENABLED,
		&chyleConfig.FEATURES.DECORATORS.GITHUBISSUE,
	}
}

func getGithubIssueDecoratorCustomValidationFuncs() []func() error {
	return []func() error{}
}

func getGithubIssueDecoratorCustomSettersFuncs() []func(*CHYLE) {
	return []func(*CHYLE){}
}

func newGithubIssueDecoratorConfigurator(config *envh.EnvTree) configurater {
	return &apiDecoratorConfigurator{
		config: config,
		apiDecoratorConfig: apiDecoratorConfig{
			"GITHUBISSUEID",
			"githubIssueId",
			"GITHUBISSUE",
			&chyleConfig.DECORATORS.GITHUBISSUE.KEYS,
			getGithubIssueDecoratorMandatoryParamsRefs(),
			getGithubIssueDecoratorFeatureRefs(),
			getGithubIssueDecoratorCustomValidationFuncs(),
			getGithubIssueDecoratorCustomSettersFuncs(),
		},
	}
}
