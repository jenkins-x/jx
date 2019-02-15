package v1

import (
	"strings"
	"testing"
)

func TestFindDuplicates(t *testing.T) {

	tests := []struct {
		name   string
		input  []string
		errors []string
	}{
		{
			name:   "Two stage name duplicated",
			input:  []string{"PipelineStructureStage 1", "PipelineStructureStage 1", "PipelineStructureStage 2", "PipelineStructureStage 2"},
			errors: []string{"PipelineStructureStage 1", "PipelineStructureStage 2"},
		}, {
			name:   "One stage name duplicated",
			input:  []string{"PipelineStructureStage 1", "PipelineStructureStage 1"},
			errors: []string{"PipelineStructureStage 1"},
		}, {
			name:   "No stage name duplicated",
			input:  []string{"PipelineStructureStage 0", "PipelineStructureStage 1", "PipelineStructureStage 2", "PipelineStructureStage 3"},
			errors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := findDuplicates(tt.input)

			if len(tt.errors) == 0 && err != nil {
				t.Fatal("Not all duplicates found A")
			}

			if len(tt.errors) > 0 && len(err.Details) > 0 {
				for _, error := range tt.errors {
					if !strings.Contains(err.Details, error) {
						t.Fatal("Not all duplicates found", error)
					}
				}
			}

			if len(tt.errors) == 0 {
				if err != nil {
					t.Fatal("Unexpected error ", err.Details)
				}
			}

		})
	}
}
