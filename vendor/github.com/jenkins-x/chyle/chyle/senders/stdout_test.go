package senders

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antham/chyle/chyle/types"
)

func TestNewStdout(t *testing.T) {
	config := stdoutConfig{FORMAT: "json"}
	assert.IsType(t, jSONStdout{}, newStdout(config))

	config = stdoutConfig{FORMAT: "template", TEMPLATE: "{{.}}"}
	assert.IsType(t, templateStdout{}, newStdout(config))
}

func TestJSONStdout(t *testing.T) {
	buf := &bytes.Buffer{}

	s := jSONStdout{buf}

	c := types.Changelog{
		Datas:     []map[string]interface{}{},
		Metadatas: map[string]interface{}{},
	}

	c.Datas = []map[string]interface{}{
		{
			"id":   1,
			"test": "test",
		},
		{
			"id":   2,
			"test": "test",
		},
	}

	err := s.Send(&c)

	assert.NoError(t, err)
	assert.Equal(t, `{"datas":[{"id":1,"test":"test"},{"id":2,"test":"test"}],"metadatas":{}}`, strings.TrimRight(buf.String(), "\n"))
}

func TestTemplateStdout(t *testing.T) {
	buf := &bytes.Buffer{}

	s := templateStdout{buf, "{{ range $key, $value := .Datas }}{{$value.id}} : {{$value.test}} | {{ end }}"}

	c := types.Changelog{
		Datas:     []map[string]interface{}{},
		Metadatas: map[string]interface{}{},
	}

	c.Datas = []map[string]interface{}{
		{
			"id":   1,
			"test": "test",
		},
		{
			"id":   2,
			"test": "test",
		},
	}

	err := s.Send(&c)

	assert.NoError(t, err)
	assert.Equal(t, `1 : test | 2 : test | `, strings.TrimRight(buf.String(), "\n"))
}
