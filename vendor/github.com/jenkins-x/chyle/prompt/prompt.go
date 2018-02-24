package prompt

import (
	"github.com/antham/strumt"
	"io"

	"github.com/antham/chyle/prompt/internal/builder"
)

// Prompts held prompts
type Prompts struct {
	prompts strumt.Prompts
}

// New creates a new prompt chain
func New(reader io.Reader, writer io.Writer) Prompts {
	return Prompts{strumt.NewPromptsFromReaderAndWriter(reader, writer)}
}

func (p *Prompts) populatePrompts(prompts []strumt.Prompter) {
	for _, item := range prompts {
		switch prompt := item.(type) {
		case strumt.LinePrompter:
			p.prompts.AddLinePrompter(prompt)
		}
	}
}

// Run starts a prompt session
func (p *Prompts) Run() builder.Store {
	store := &builder.Store{}
	prompts := mergePrompters(
		newMainMenu(),
		newMandatoryOption(store),
		newMatchers(store),
		newExtractors(store),
		newDecorators(store),
		newSenders(store),
	)

	p.populatePrompts(prompts)

	p.prompts.SetFirst("referenceFrom")
	p.prompts.Run()

	return *store
}
