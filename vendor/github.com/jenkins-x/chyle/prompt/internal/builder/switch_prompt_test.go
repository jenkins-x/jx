package builder

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antham/strumt"
)

func TestCreateSwitchChoice(t *testing.T) {
	sws := []strumt.Prompter{
		&switchPrompt{
			"test",
			[]SwitchConfig{
				{
					"1",
					"1 - Choice number 1",
					"test",
				},
				{
					"2",
					"2 - Choice number 2",
					"test",
				},
				{
					"3",
					"3 - Choice number 3",
					"",
				},
			},
		},
	}

	var stdout bytes.Buffer
	buf := "1\n2\n4\n3\n"

	p := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	for _, sw := range sws {
		p.AddLinePrompter(sw.(strumt.LinePrompter))
	}

	p.SetFirst("test")
	p.Run()

	scenario := p.Scenario()

	steps := []struct {
		input string
		err   error
	}{
		{
			"1",
			nil,
		},
		{
			"2",
			nil,
		},
		{
			"4",
			fmt.Errorf("This choice doesn't exist"),
		},
		{
			"3",
			nil,
		},
	}

	for i, step := range steps {
		assert.Len(t, scenario[i].Inputs(), 1)
		assert.Equal(t, scenario[i].Inputs()[0], step.input)
		assert.Equal(t, scenario[i].Error(), step.err)
	}
}
