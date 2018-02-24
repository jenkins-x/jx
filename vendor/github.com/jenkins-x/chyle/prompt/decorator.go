package prompt

import (
	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"
)

func newDecorators(store *builder.Store) []strumt.Prompter {
	return mergePrompters(
		decorator,
		newCustomAPIDecorator(store),
		newJiraIssueDecorator(store),
		newGithubIssueDecorator(store),
		newShellDecorator(store),
		newEnvDecorator(store),
	)
}

var decorator = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"decoratorChoice",
		addMainMenuAndQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a custom api decorator",
					NextPromptID: "extractorOrigKeyCustomAPI",
				},
				{
					Choice:       "2",
					PromptString: "Add a jira issue decorator",
					NextPromptID: "extractorOrigKeyJiraIssueID",
				},
				{
					Choice:       "3",
					PromptString: "Add a github issue decorator",
					NextPromptID: "extractorOrigKeyGithubIssueID",
				},
				{
					Choice:       "4",
					PromptString: "Add a shell decorator",
					NextPromptID: "shellDecoratorCommand",
				},
				{
					Choice:       "5",
					PromptString: "Add an environment variable decorator to add an environment variable to the global metadata hashmap",
					NextPromptID: "envDecoratorVarName",
				},
			},
		),
	),
}

func newCustomAPIDecorator(store *builder.Store) []strumt.Prompter {
	return mergePrompters(
		builder.NewGroupEnvPromptWithCounter(customAPIDecoratorKeys, store),
		builder.NewEnvPrompts(customAPIDecorator, store),
		customAPIDecoratorChoice,
	)
}

var customAPIDecorator = []builder.EnvConfig{
	{
		ID:                  "extractorOrigKeyCustomAPI",
		NextID:              "extractorDestKeyCustomAPI",
		Env:                 "CHYLE_EXTRACTORS_CUSTOMAPIID_ORIGKEY",
		PromptString:        "Enter a commit field where your custom api id is located",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorDestKeyCustomAPI",
		NextID:              "extractorRegCustomAPI",
		Env:                 "CHYLE_EXTRACTORS_CUSTOMAPIID_DESTKEY",
		PromptString:        "Enter a name for the key which will receive the extracted value",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorRegCustomAPI",
		NextID:              "customAPIDecoratorEndpoint",
		Env:                 "CHYLE_EXTRACTORS_CUSTOMAPIID_REG",
		PromptString:        "Enter a regexp to extract custom api id",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "customAPIDecoratorEndpoint",
		NextID:              "customAPIDecoratorToken",
		Env:                 "CHYLE_DECORATORS_CUSTOMAPIID_ENDPOINT_URL",
		PromptString:        "Enter custom api endpoint URL, use {{ID}} as a placeholder to interpolate the id you extracted before in URL if you need to",
		Validator:           validateURL,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "customAPIDecoratorToken",
		NextID:              "customAPIDecoratorDestKey",
		Env:                 "CHYLE_DECORATORS_CUSTOMAPIID_CREDENTIALS_TOKEN",
		PromptString:        "Enter token submitted as authorization header when calling your api",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

var customAPIDecoratorKeys = []builder.EnvConfig{
	{
		ID:                  "customAPIDecoratorDestKey",
		NextID:              "customAPIDecoratorField",
		Env:                 "CHYLE_DECORATORS_CUSTOMAPIID_KEYS_*_DESTKEY",
		PromptString:        "A name for the key which will receive the data extracted from the custom api",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "customAPIDecoratorField",
		NextID:              "customAPIDecoratorChoice",
		Env:                 "CHYLE_DECORATORS_CUSTOMAPIID_KEYS_*_FIELD",
		PromptString:        `The field to extract from your custom api response payload, use dot notation to extract a nested value (eg: "fields.summary")`,
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

var customAPIDecoratorChoice = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"customAPIDecoratorChoice",
		addMainMenuAndQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a new custom api decorator field",
					NextPromptID: "customAPIDecoratorDestKey",
				},
			},
		),
	),
}

func newJiraIssueDecorator(store *builder.Store) []strumt.Prompter {
	return mergePrompters(
		builder.NewGroupEnvPromptWithCounter(jiraIssueDecoratorKeys, store),
		builder.NewEnvPrompts(jiraIssueDecorator, store),
		jiraIssueDecoratorChoice,
	)
}

var jiraIssueDecorator = []builder.EnvConfig{
	{
		ID:                  "extractorOrigKeyJiraIssueID",
		NextID:              "extractorRegJiraIssueID",
		Env:                 "CHYLE_EXTRACTORS_JIRAISSUEID_ORIGKEY",
		PromptString:        "Enter a commit field where your jira issue id is located",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorRegJiraIssueID",
		NextID:              "jiraIssueDecoratorEndpoint",
		Env:                 "CHYLE_EXTRACTORS_JIRAISSUEID_REG",
		PromptString:        "Enter a regexp to extract jira issue id",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "jiraIssueDecoratorEndpoint",
		NextID:              "jiraIssueDecoratorUsername",
		Env:                 "CHYLE_DECORATORS_JIRAISSUE_ENDPOINT_URL",
		PromptString:        "Enter jira api endpoint URL",
		Validator:           validateURL,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "jiraIssueDecoratorUsername",
		NextID:              "jiraIssueDecoratorPassword",
		Env:                 "CHYLE_DECORATORS_JIRAISSUE_CREDENTIALS_USERNAME",
		PromptString:        "Enter jira username",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:           "jiraIssueDecoratorPassword",
		NextID:       "jiraIssueDecoratorDestKey",
		Env:          "CHYLE_DECORATORS_JIRAISSUE_CREDENTIALS_PASSWORD",
		PromptString: "Enter jira password",
		Validator:    validateDefinedValue,
		RunBeforeNextPrompt: func(value string, store *builder.Store) {
			(*store)["CHYLE_EXTRACTORS_JIRAISSUEID_DESTKEY"] = "jiraIssueId"
		},
	},
}

var jiraIssueDecoratorKeys = []builder.EnvConfig{
	{
		ID:                  "jiraIssueDecoratorDestKey",
		NextID:              "jiraIssueDecoratorField",
		Env:                 "CHYLE_DECORATORS_JIRAISSUE_KEYS_*_DESTKEY",
		PromptString:        "A name for the key which will receive the extracted value",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "jiraIssueDecoratorField",
		NextID:              "jiraIssueDecoratorChoice",
		Env:                 "CHYLE_DECORATORS_JIRAISSUE_KEYS_*_FIELD",
		PromptString:        `The field to extract from jira api response payload, use dot notation to extract a nested value (eg: "fields.summary" will retrieve the summary of a jira issue)`,
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

var jiraIssueDecoratorChoice = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"jiraIssueDecoratorChoice",
		addMainMenuAndQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a new jira issue decorator field",
					NextPromptID: "jiraIssueDecoratorDestKey",
				},
			},
		),
	),
}

func newGithubIssueDecorator(store *builder.Store) []strumt.Prompter {
	return mergePrompters(
		builder.NewGroupEnvPromptWithCounter(githubIssueDecoratorKeys, store),
		builder.NewEnvPrompts(githubIssueDecorator, store),
		githubIssueDecoratorChoice,
	)
}

var githubIssueDecorator = []builder.EnvConfig{
	{
		ID:                  "extractorOrigKeyGithubIssueID",
		NextID:              "extractorRegGithubIssueID",
		Env:                 "CHYLE_EXTRACTORS_GITHUBISSUEID_ORIGKEY",
		PromptString:        "Enter a commit field where your github issue id is located",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorRegGithubIssueID",
		NextID:              "githubIssueCredentialsToken",
		Env:                 "CHYLE_EXTRACTORS_GITHUBISSUEID_REG",
		PromptString:        "Enter a regexp to extract github issue id",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "githubIssueCredentialsToken",
		NextID:              "githubIssueCredentialsOwner",
		Env:                 "CHYLE_DECORATORS_GITHUBISSUE_CREDENTIALS_OAUTHTOKEN",
		PromptString:        "Enter github oauth token",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:           "githubIssueCredentialsOwner",
		NextID:       "githubIssueDecoratorDestKey",
		Env:          "CHYLE_DECORATORS_GITHUBISSUE_CREDENTIALS_OWNER",
		PromptString: "Enter github owner name",
		Validator:    validateDefinedValue,
		RunBeforeNextPrompt: func(value string, store *builder.Store) {
			(*store)["CHYLE_EXTRACTORS_GITHUBISSUEID_DESTKEY"] = "githubIssueId"
		},
	},
}

var githubIssueDecoratorKeys = []builder.EnvConfig{
	{
		ID:                  "githubIssueDecoratorDestKey",
		NextID:              "githubIssueDecoratorField",
		Env:                 "CHYLE_DECORATORS_GITHUBISSUE_KEYS_*_DESTKEY",
		PromptString:        "A name for the key which will receive the extracted value",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "githubIssueDecoratorField",
		NextID:              "githubIssueDecoratorChoice",
		Env:                 "CHYLE_DECORATORS_GITHUBISSUE_KEYS_*_FIELD",
		PromptString:        `The field to extract from github issue api response payload, use dot notation to extract a nested value (eg: "user.id" will retrieve the user who created the issue)`,
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

var githubIssueDecoratorChoice = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"githubIssueDecoratorChoice",
		addMainMenuAndQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a new github issue decorator field",
					NextPromptID: "githubIssueDecoratorDestKey",
				},
			},
		),
	),
}

func newShellDecorator(store *builder.Store) []strumt.Prompter {
	return builder.NewGroupEnvPromptWithCounter(shellDecoratorKeys, store)
}

var shellDecoratorKeys = []builder.EnvConfig{
	{
		ID:                  "shellDecoratorCommand",
		NextID:              "shellDecoratorOrigKey",
		Env:                 "CHYLE_DECORATORS_SHELL_*_COMMAND",
		PromptString:        "Shell command to execute",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "shellDecoratorOrigKey",
		NextID:              "shellDecoratorDestKey",
		Env:                 "CHYLE_DECORATORS_SHELL_*_ORIGKEY",
		PromptString:        "A field from which you want to use the content to pipe a command on",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "shellDecoratorDestKey",
		NextID:              "decoratorChoice",
		Env:                 "CHYLE_DECORATORS_SHELL_*_DESTKEY",
		PromptString:        "A name for the key which will receive the extracted value",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

func newEnvDecorator(store *builder.Store) []strumt.Prompter {
	return builder.NewGroupEnvPromptWithCounter(envDecoratorKeys, store)
}

var envDecoratorKeys = []builder.EnvConfig{
	{
		ID:                  "envDecoratorVarName",
		NextID:              "envDecoratorDestKey",
		Env:                 "CHYLE_DECORATORS_ENV_*_VARNAME",
		PromptString:        "Environment variable name to dump in metadatas",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "envDecoratorDestKey",
		NextID:              "decoratorChoice",
		Env:                 "CHYLE_DECORATORS_ENV_*_DESTKEY",
		PromptString:        "The name of the key where to store the dumped value in metadatas",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}
