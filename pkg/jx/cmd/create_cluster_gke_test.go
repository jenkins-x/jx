package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sanitizeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		want     string
	}{
		{"Replaces . in username for -", "test.person", "test-person"},
		{"Replaces _ in username for -", "test_person", "test-person"},
		{"Replaces uppercase in username for lowercase", "Test", "test"},
		{"Doesn't do anything for empty user names", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, sanitizeLabel(tt.username), tt.want)
		})
	}
}
