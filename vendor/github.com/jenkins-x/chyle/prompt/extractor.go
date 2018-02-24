package prompt

import (
	"fmt"

	"github.com/antham/strumt"

	"github.com/antham/chyle/prompt/internal/builder"
)

func newExtractors(store *builder.Store) []strumt.Prompter {
	return builder.NewGroupEnvPromptWithCounter(extractor, store)
}

var extractor = []builder.EnvConfig{
	{
		ID:                  "extractorOrigKey",
		NextID:              "extractorDestKey",
		Env:                 "CHYLE_EXTRACTORS_*_ORIGKEY",
		PromptString:        "Enter a commit field from which we want to extract datas (id, authorName, authorEmail, authorDate, committerName, committerEmail, committerMessage, type)",
		Validator:           validateExtractorCommitField,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorDestKey",
		NextID:              "extractorReg",
		Env:                 "CHYLE_EXTRACTORS_*_DESTKEY",
		PromptString:        "Enter a name for the key which will receive the extracted value",
		Validator:           validateDefinedValue,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
	{
		ID:                  "extractorReg",
		NextID:              "mainMenu",
		Env:                 "CHYLE_EXTRACTORS_*_REG",
		PromptString:        "Enter a regexp used to extract a data",
		Validator:           validateRegexp,
		RunBeforeNextPrompt: noOpRunBeforeNextPrompt,
	},
}

func validateExtractorCommitField(value string) error {
	fields := []string{"id", "authorName", "authorEmail", "authorDate", "committerName", "committerEmail", "committerMessage", "type"}

	for _, f := range fields {
		if value == f {
			return nil
		}
	}

	return fmt.Errorf("Must be one of %v", fields)
}
