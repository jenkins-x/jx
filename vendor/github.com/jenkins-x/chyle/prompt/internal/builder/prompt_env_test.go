package builder

import (
	"bytes"
	"errors"
	"testing"

	"github.com/antham/strumt"
	"github.com/stretchr/testify/assert"
)

func TestNewEnvPrompt(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "1\n"
	p := NewEnvPrompt(EnvConfig{"TEST", "NEXT_TEST", "TEST_NEW_ENV_PROMPT", "Enter a value", func(value string) error { return nil }, "", func(value string, store *Store) {}}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	s.AddLinePrompter(p.(strumt.LinePrompter))
	s.SetFirst("TEST")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 1)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "1")
	assert.Nil(t, scenario[0].Error())

	assert.Equal(t, &Store{"TEST_NEW_ENV_PROMPT": "1"}, store)
}

func TestNewEnvPromptWithAnEmptyValueAndNoValidationRules(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "\n"
	p := NewEnvPrompt(EnvConfig{"TEST", "NEXT_TEST", "TEST_NEW_ENV_PROMPT", "Enter a value", func(value string) error { return nil }, "", func(value string, store *Store) {}}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	s.AddLinePrompter(p.(strumt.LinePrompter))
	s.SetFirst("TEST")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 1)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "")
	assert.Nil(t, scenario[0].Error())

	assert.Equal(t, &Store{}, store)
}

func TestNewEnvPromptWithAnEmptyValueAndValidationRulesAndDefaultValue(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "\n"
	p := NewEnvPrompt(EnvConfig{"TEST", "NEXT_TEST", "TEST_NEW_ENV_PROMPT", "Enter a value", func(value string) error { return errors.New("An error occured") }, "DEFAULT_VALUE", func(value string, store *Store) {}}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	s.AddLinePrompter(p.(strumt.LinePrompter))
	s.SetFirst("TEST")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 1)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "")
	assert.Nil(t, scenario[0].Error())

	assert.Equal(t, &Store{"TEST_NEW_ENV_PROMPT": "DEFAULT_VALUE"}, store)
}

func TestNewEnvPromptWithAPromptHook(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "TEST\n"
	p := NewEnvPrompt(EnvConfig{"TEST", "NEXT_TEST", "TEST_NEW_ENV_PROMPT", "Enter a value", func(value string) error { return nil }, "", func(value string, store *Store) { (*store)["TEST_NEW_ENV_PROMPT_2"] = "TEST_2" }}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	s.AddLinePrompter(p.(strumt.LinePrompter))
	s.SetFirst("TEST")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 1)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "TEST")
	assert.Nil(t, scenario[0].Error())

	assert.Equal(t, &Store{"TEST_NEW_ENV_PROMPT": "TEST", "TEST_NEW_ENV_PROMPT_2": "TEST_2"}, store)
}

func TestNewEnvPromptWithEmptyValueAndCustomErrorGiven(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "\nfalse\ntrue\n"
	p := NewEnvPrompt(EnvConfig{"TEST", "NEXT_TEST", "TEST_NEW_ENV_PROMPT", "Enter a value", func(value string) error {
		if value != "true" {
			return errors.New("Value must be true")
		}
		return nil
	}, "", func(value string, store *Store) {}}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	s.AddLinePrompter(p.(strumt.LinePrompter))
	s.SetFirst("TEST")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 3)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "")
	assert.EqualError(t, scenario[0].Error(), "Value must be true")
	assert.Equal(t, scenario[1].PromptString(), "Enter a value")
	assert.Len(t, scenario[1].Inputs(), 1)
	assert.Equal(t, scenario[1].Inputs()[0], "false")
	assert.Equal(t, scenario[1].Error().Error(), "Value must be true")
	assert.Equal(t, scenario[2].PromptString(), "Enter a value")
	assert.Len(t, scenario[2].Inputs(), 1)
	assert.Equal(t, scenario[2].Inputs()[0], "true")
	assert.Nil(t, scenario[2].Error())

	assert.Equal(t, &Store{"TEST_NEW_ENV_PROMPT": "true"}, store)
}

func TestNewEnvPrompts(t *testing.T) {
	store := &Store{}

	var stdout bytes.Buffer
	buf := "1\n2\n"
	p := NewEnvPrompts([]EnvConfig{
		{"TEST1", "TEST2", "TEST_PROMPT_1", "Enter a value for prompt 1", func(value string) error { return nil }, "", func(value string, store *Store) {}},
		{"TEST2", "", "TEST_PROMPT_2", "Enter a value for prompt 2", func(value string) error { return nil }, "", func(value string, store *Store) {}},
	}, store)

	s := strumt.NewPromptsFromReaderAndWriter(bytes.NewBufferString(buf), &stdout)
	for _, item := range p {
		switch prompt := item.(type) {
		case strumt.LinePrompter:
			s.AddLinePrompter(prompt)
		case strumt.MultilinePrompter:
			s.AddMultilinePrompter(prompt)
		}
	}
	s.SetFirst("TEST1")
	s.Run()

	scenario := s.Scenario()

	assert.Len(t, scenario, 2)
	assert.Equal(t, scenario[0].PromptString(), "Enter a value for prompt 1")
	assert.Len(t, scenario[0].Inputs(), 1)
	assert.Equal(t, scenario[0].Inputs()[0], "1")
	assert.Nil(t, scenario[0].Error())
	assert.Equal(t, scenario[1].PromptString(), "Enter a value for prompt 2")
	assert.Len(t, scenario[1].Inputs(), 1)
	assert.Equal(t, scenario[1].Inputs()[0], "2")
	assert.Nil(t, scenario[1].Error())

	assert.Equal(t, &Store{"TEST_PROMPT_1": "1", "TEST_PROMPT_2": "2"}, store)
}
