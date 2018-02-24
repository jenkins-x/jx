package prompt

import (
	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"
)

func newMandatoryOption(store *builder.Store) []strumt.Prompter {
	return builder.NewEnvPrompts(mandatoryOption, store)
}

var mandatoryOption = []builder.EnvConfig{
	{
		ID:                  "referenceFrom",
		NextID:              "referenceTo",
		Env:                 "CHYLE_GIT_REFERENCE_FROM",
		PromptString:        "Enter a git commit ID that start your range, this value will likely vary if you want to integrate it to a CI tool so you will need to generate this value according to your context",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "referenceTo",
		NextID:              "gitPath",
		Env:                 "CHYLE_GIT_REFERENCE_TO",
		PromptString:        "Enter a git commit ID that finish your range, this value will likely vary if you want to integrate it to a CI tool so you will need to generate this value according to your context",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "gitPath",
		NextID:              "mainMenu",
		Env:                 "CHYLE_GIT_REPOSITORY_PATH",
		PromptString:        "Enter the location of your git path repository",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}
