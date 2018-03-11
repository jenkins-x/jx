package survey

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestSelectRender(t *testing.T) {

	prompt := Select{
		Message: "Pick your word:",
		Options: []string{"foo", "bar", "baz", "buz"},
		Default: "baz",
	}

	helpfulPrompt := prompt
	helpfulPrompt.Help = "This is helpful"

	tests := []struct {
		title    string
		prompt   Select
		data     SelectTemplateData
		expected string
	}{
		{
			"Test Select question output",
			prompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			`? Pick your word:  [Use arrows to move, type to filter]
  foo
  bar
❯ baz
  buz
`,
		},
		{
			"Test Select answer output",
			prompt,
			SelectTemplateData{Answer: "buz", ShowAnswer: true, PageEntries: prompt.Options},
			"? Pick your word: buz\n",
		},
		{
			"Test Select question output with help hidden",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, PageEntries: prompt.Options},
			`? Pick your word:  [Use arrows to move, type to filter, ? for more help]
  foo
  bar
❯ baz
  buz
`,
		},
		{
			"Test Select question output with help shown",
			helpfulPrompt,
			SelectTemplateData{SelectedIndex: 2, ShowHelp: true, PageEntries: prompt.Options},
			`ⓘ This is helpful
? Pick your word:  [Use arrows to move, type to filter]
  foo
  bar
❯ baz
  buz
`,
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.Select = test.prompt
		err := test.prompt.Render(
			SelectQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
