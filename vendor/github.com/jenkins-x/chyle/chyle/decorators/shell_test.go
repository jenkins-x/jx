package decorators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShell(t *testing.T) {
	tests := []struct {
		config   shellConfig
		datas    map[string]interface{}
		expected map[string]interface{}
		errStr   string
	}{
		{
			shellConfig{
				"TEST": {
					`tr -s 'x' 'y'`,
					"FROM",
					"TO",
				},
			},
			map[string]interface{}{
				"FROM": `a sentence with the letter "x"`,
			},
			map[string]interface{}{
				"FROM": `a sentence with the letter "x"`,
				"TO":   `a sentence with the letter "y"`,
			},
			"",
		},
		{
			shellConfig{
				"TEST": {
					`tr -s 'x' 'y'`,
					"FROM",
					"TO",
				},
			},
			map[string]interface{}{
				"FROM": "",
			},
			map[string]interface{}{
				"FROM": "",
				"TO":   "",
			},
			"",
		},
		{
			shellConfig{
				"TEST": {
					`sed -s "s/whatever/world/"`,
					"FROM",
					"TO",
				},
			},
			map[string]interface{}{
				"FROM": `hello "whatever" !`,
			},
			map[string]interface{}{
				"FROM": `hello "whatever" !`,
				"TO":   `hello "world" !`,
			},
			"",
		},
		{
			shellConfig{
				"TEST": {
					`whatever`,
					"FROM",
					"TO",
				},
			},
			map[string]interface{}{
				"FROM": "",
			},
			map[string]interface{}{
				"FROM": "",
				"TO":   "",
			},
			`echo ""|whatever : command failed`,
		},
	}

	for _, test := range tests {
		s := newShell(test.config)

		datas, err := s[0].Decorate(&test.datas)

		if test.errStr != "" {
			assert.EqualError(t, err, test.errStr)
		}

		assert.Equal(t, test.expected, *datas)
	}
}
