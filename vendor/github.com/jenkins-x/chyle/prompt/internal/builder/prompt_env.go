package builder

import (
	"github.com/antham/strumt"
)

// NewEnvPrompts creates several prompts at once to populate environments variables
func NewEnvPrompts(configs []EnvConfig, store *Store) []strumt.Prompter {
	results := []strumt.Prompter{}

	for _, config := range configs {
		results = append(results, NewEnvPrompt(config, store))
	}

	return results
}

// NewEnvPrompt creates a prompt to populate an environment variable
func NewEnvPrompt(config EnvConfig, store *Store) strumt.Prompter {
	return &GenericPrompt{
		config.ID,
		config.PromptString,
		func(value string) string {
			config.RunBeforeNextPrompt(value, store)

			return config.NextID
		},
		func(error) string { return config.ID },
		ParseEnv(config.Validator, config.Env, config.DefaultValue, store),
	}
}

// ParseEnv provides an env parser callback
func ParseEnv(validator func(string) error, env string, defaultValue string, store *Store) func(value string) error {
	return func(value string) error {
		if value == "" && defaultValue != "" {
			(*store)[env] = defaultValue

			return nil
		}

		if err := validator(value); err != nil {
			return err
		}

		if value != "" {
			(*store)[env] = value
		}

		return nil
	}
}
