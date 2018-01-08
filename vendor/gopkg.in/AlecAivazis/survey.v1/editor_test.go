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

func TestEditorRender(t *testing.T) {
	tests := []struct {
		title    string
		prompt   Editor
		data     EditorTemplateData
		expected string
	}{
		{
			"Test Editor question output without default",
			Editor{Message: "What is your favorite month:"},
			EditorTemplateData{},
			"? What is your favorite month: [Enter to launch editor] ",
		},
		{
			"Test Editor question output with default",
			Editor{Message: "What is your favorite month:", Default: "April"},
			EditorTemplateData{},
			"? What is your favorite month: (April) [Enter to launch editor] ",
		},
		{
			"Test Editor question output with HideDefault",
			Editor{Message: "What is your favorite month:", Default: "April", HideDefault: true},
			EditorTemplateData{},
			"? What is your favorite month: [Enter to launch editor] ",
		},
		{
			"Test Editor answer output",
			Editor{Message: "What is your favorite month:"},
			EditorTemplateData{Answer: "October", ShowAnswer: true},
			"? What is your favorite month: October\n",
		},
		{
			"Test Editor question output without default but with help hidden",
			Editor{Message: "What is your favorite month:", Help: "This is helpful"},
			EditorTemplateData{},
			"? What is your favorite month: [? for help] [Enter to launch editor] ",
		},
		{
			"Test Editor question output with default and with help hidden",
			Editor{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			EditorTemplateData{},
			"? What is your favorite month: [? for help] (April) [Enter to launch editor] ",
		},
		{
			"Test Editor question output without default but with help shown",
			Editor{Message: "What is your favorite month:", Help: "This is helpful"},
			EditorTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: [Enter to launch editor] `,
		},
		{
			"Test Editor question output with default and with help shown",
			Editor{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			EditorTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: (April) [Enter to launch editor] `,
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.Editor = test.prompt
		err := test.prompt.Render(
			EditorQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
