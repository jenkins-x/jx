package syntax

import (
	"io/ioutil"
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
			input:  []string{"Stage 1", "Stage 1", "Stage 2", "Stage 2"},
			errors: []string{"Stage 1", "Stage 2"},
		}, {
			name:   "One stage name duplicated",
			input:  []string{"Stage 1", "Stage 1"},
			errors: []string{"Stage 1"},
		}, {
			name:   "No stage name duplicated",
			input:  []string{"Stage 0", "Stage 1", "Stage 2", "Stage 3"},
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

func TestFindDuplicatesWithStages(t *testing.T) {
	tests := []struct {
		name          string
		expectedError []string
	}{
		{
			name:          "stages_names_ok.yaml",
			expectedError: []string{},
		},
		{
			name:          "stages_names_ok_with_sub_stages.yaml",
			expectedError: []string{"Duplicate stage name 'A Working title 1'", "Duplicate stage name 'A Working title 2'"},
		},
		{
			name:          "stages_names_duplicates_with_sub_stages.yaml",
			expectedError: []string{"Duplicate stage name 'Stage With Stages'"},
		},
		{
			name:          "stages_names_duplicates.yaml",
			expectedError: []string{"Duplicate stage name 'A Working Stage'"},
		},
		{
			name:          "stages_names_with_sub_stages.yaml",
			expectedError: []string{"Duplicate stage name 'A Working title 2'", "Duplicate stage name 'A Working title'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := "test_data/stage_name_validation/" + tt.name
			file, err := ioutil.ReadFile(fileName)

			if err != nil {
				println("ERROR: Couldn't read file ", fileName, " with error ", err)
			}

			yaml := string(file)
			parsed, _ := ParseJenkinsfileYaml(yaml)

			error := validateStageNames(parsed)

			if len(tt.expectedError) == 0 {
				if len(error.Error()) > 0 {
					t.Fatal("Unexpected error ", error.Error())
				}
			}
			for _, expected := range tt.expectedError {
				if !strings.Contains(error.Error(), expected) {
					t.Fatal("missing  expected error '", expected, "'")
				}

			}
		})
	}
}
