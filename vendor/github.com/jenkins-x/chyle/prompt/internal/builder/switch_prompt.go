package builder

import (
	"github.com/antham/strumt"

	"fmt"
)

// NewSwitchPrompt creates a new prompt used to provides several choices, like a menu can do
func NewSwitchPrompt(ID string, choices []SwitchConfig) strumt.Prompter {
	return &switchPrompt{ID, choices}
}

// SwitchConfig provides a configuration to switch prompt
type SwitchConfig struct {
	Choice       string
	PromptString string
	NextPromptID string
}

type switchPrompt struct {
	iD      string
	choices []SwitchConfig
}

func (s *switchPrompt) ID() string {
	return s.iD
}

func (s *switchPrompt) PromptString() string {
	out := fmt.Sprintf("Choose one of this option and press enter:\n")

	for _, choice := range s.choices {
		out += fmt.Sprintf("%s - %s\n", choice.Choice, choice.PromptString)
	}

	return out
}

func (s *switchPrompt) Parse(value string) error {
	if value == "" {
		return fmt.Errorf("No value given")
	}

	for _, choice := range s.choices {
		if choice.Choice == value {
			return nil
		}
	}

	return fmt.Errorf("This choice doesn't exist")
}

func (s *switchPrompt) NextOnSuccess(value string) string {
	for _, choice := range s.choices {
		if choice.Choice == value {
			return choice.NextPromptID
		}
	}

	return ""
}

func (s *switchPrompt) NextOnError(err error) string {
	return s.iD
}
