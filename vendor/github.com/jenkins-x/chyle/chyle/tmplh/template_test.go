package tmplh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPopulateTemplate(t *testing.T) {
	tests := []struct {
		ID       string
		template string
		data     interface{}
		expected string
		errStr   string
	}{
		{
			"test",
			"{{.test}}",
			map[string]string{"test": "Hello world !"},
			"Hello world !",
			``,
		},
		{
			"test",
			"{{.test",
			map[string]string{"test": "Hello world !"},
			"",
			`check your template is well-formed : template: test:1: unclosed action`,
		},
		{
			"test",
			`{{ upper "hello" }}`,
			``,
			"HELLO",
			``,
		},
		{
			"test",
			`{{ set "test" "whatever" }}{{ get "test" }}`,
			``,
			`whatever`,
			``,
		},
		{
			"test",
			`{{ set "test" true }}{{ get "test" }}`,
			``,
			`true`,
			``,
		},
		{
			"test",
			`{{ set "test" 1 }}{{ get "test" }}`,
			``,
			`1`,
			``,
		},
		{
			"test",
			`{{ set "test" "whatever" }}{{ if isset "test" }}{{ get "test" }}{{ end }}`,
			``,
			`whatever`,
			``,
		},
		{
			"test",
			`{{ if isset "test" }}{{ get "test" }}{{ end }}`,
			``,
			``,
			``,
		},
	}

	for _, test := range tests {
		store = map[string]interface{}{}

		d, err := Build(test.ID, test.template, test.data)

		if err != nil {
			assert.EqualError(t, err, test.errStr)

			continue
		}

		assert.Equal(t, test.expected, d)
	}
}
