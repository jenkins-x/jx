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

func TestInputRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Input
		data     InputTemplateData
		expected string
	}{
		{
			"Test Input question output without default",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{},
			"? What is your favorite month: ",
		},
		{
			"Test Input question output with default",
			Input{Message: "What is your favorite month:", Default: "April"},
			InputTemplateData{},
			"? What is your favorite month: (April) ",
		},
		{
			"Test Input answer output",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{Answer: "October", ShowAnswer: true},
			"? What is your favorite month: October\n",
		},
		{
			"Test Input question output without default but with help hidden",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{},
			"? What is your favorite month: [? for help] ",
		},
		{
			"Test Input question output with default and with help hidden",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{},
			"? What is your favorite month: [? for help] (April) ",
		},
		{
			"Test Input question output without default but with help shown",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: `,
		},
		{
			"Test Input question output with default and with help shown",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: (April) `,
		},
	}

	outputBuffer := bytes.NewBufferString("")
	terminal.Stdout = outputBuffer

	for _, test := range tests {
		outputBuffer.Reset()
		test.data.Input = test.prompt
		err := test.prompt.Render(
			InputQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, outputBuffer.String(), test.title)
	}
}
