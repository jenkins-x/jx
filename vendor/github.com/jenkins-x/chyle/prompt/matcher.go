package prompt

import (
	"fmt"

	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"
)

func newMatchers(store *builder.Store) []strumt.Prompter {
	return mergePrompters(
		matcherChoice,
		builder.NewEnvPrompts(matcher, store),
	)
}

var matcherChoice = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"matcherChoice",
		addMainMenuAndQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a type matcher, it's used to match merge commit or regular one",
					NextPromptID: "matcherType",
				},
				{
					Choice:       "2",
					PromptString: "Add a message matcher, it's used to match a commit according to a pattern found in commit message",
					NextPromptID: "matcherMessage",
				},
				{
					Choice:       "3",
					PromptString: "Add a committer matcher, it's used to match a commit according to a pattern apply to the committer field",
					NextPromptID: "matcherCommitter",
				},
				{
					Choice:       "4",
					PromptString: "Add an author matcher, it's used to match a commit according to a pattern apply to the author field",
					NextPromptID: "matcherAuthor",
				},
			},
		),
	),
}

var matcher = []builder.EnvConfig{
	{
		ID:                  "matcherType",
		NextID:              "matcherChoice",
		Env:                 "CHYLE_MATCHERS_TYPE",
		PromptString:        "Enter a matcher type (regular or merge)",
		Validator:           validateMatcherType,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "matcherMessage",
		NextID:              "matcherChoice",
		Env:                 "CHYLE_MATCHERS_MESSAGE",
		PromptString:        "Enter a regexp to match commit message",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "matcherCommitter",
		NextID:              "matcherChoice",
		Env:                 "CHYLE_MATCHERS_COMMITTER",
		PromptString:        "Enter a regexp to match git committer",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "matcherAuthor",
		NextID:              "matcherChoice",
		Env:                 "CHYLE_MATCHERS_AUTHOR",
		PromptString:        "Enter a regexp to match git author",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

func validateMatcherType(value string) error {
	if value != "regular" && value != "merge" {
		return fmt.Errorf(`Must be "regular" or "merge"`)
	}

	return nil
}
