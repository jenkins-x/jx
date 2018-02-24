package senders

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antham/chyle/chyle/types"
)

func TestSend(t *testing.T) {
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

	err := Send(&[]Sender{s}, &c)

	assert.NoError(t, err)
	assert.Equal(t, `{"datas":[{"id":1,"test":"test"},{"id":2,"test":"test"}],"metadatas":{}}`, strings.TrimRight(buf.String(), "\n"))
}
func TestCreate(t *testing.T) {
	tests := []func() (Features, Config){
		func() (Features, Config) {
			config := stdoutConfig{}
			config.FORMAT = "json"

			return Features{ENABLED: true, STDOUT: true}, Config{STDOUT: config}
		},
		func() (Features, Config) {
			config := githubReleaseConfig{}
			config.CREDENTIALS.OAUTHTOKEN = "test"
			config.CREDENTIALS.OWNER = "test"
			config.RELEASE.TAGNAME = "test"
			config.RELEASE.TEMPLATE = "test"
			config.REPOSITORY.NAME = "test"

			return Features{ENABLED: true, GITHUBRELEASE: true}, Config{GITHUBRELEASE: config}
		},
		func() (Features, Config) {
			config := customAPIConfig{}
			config.CREDENTIALS.TOKEN = "test"
			config.ENDPOINT.URL = "http://test.com"

			return Features{ENABLED: true, CUSTOMAPI: true}, Config{CUSTOMAPI: config}
		},
	}

	for _, f := range tests {
		features, config := f()

		s := Create(features, config)

		assert.Len(t, *s, 1)
	}
}

func TestCreateWithFeatureDisabled(t *testing.T) {
	s := Create(Features{STDOUT: true}, Config{STDOUT: stdoutConfig{FORMAT: "json"}})

	assert.Len(t, *s, 0)
}
