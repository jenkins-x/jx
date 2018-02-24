package prompt

import (
	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"
)

var mainMenu = []strumt.Prompter{
	builder.NewSwitchPrompt(
		"mainMenu",
		addQuitChoice(
			[]builder.SwitchConfig{
				{
					Choice:       "1",
					PromptString: "Add a matcher, they are used to filters git commit according to various criterias (message pattern, author, and so on)",
					NextPromptID: "matcherChoice",
				},
				{
					Choice:       "2",
					PromptString: "Add an extractor, they are used to extract datas from commit fields, for instance extracting PR number from a commit message",
					NextPromptID: "extractorOrigKey",
				},
				{
					Choice:       "3",
					PromptString: "Add a decorator, they are used to add datas from external sources, for instance contacting github issue api to add ticket title",
					NextPromptID: "decoratorChoice",
				},
				{
					Choice:       "4",
					PromptString: "Add a sender, they are used to send the result to an external source, for instance dumping the result as markdown on stdout",
					NextPromptID: "senderChoice",
				},
			},
		),
	),
}

func newMainMenu() []strumt.Prompter {
	return mainMenu
}
