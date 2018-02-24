package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingEnvError(t *testing.T) {
	e := MissingEnvError{envs: []string{"TEST"}}

	assert.Equal(t, `environment variable missing : "TEST"`, e.Error())

	assert.Equal(t, []string{"TEST"}, e.Envs())

	e = MissingEnvError{envs: []string{"TEST", "TEST1"}}

	assert.Equal(t, `environments variables missing : "TEST", "TEST1"`, e.Error())

	assert.Equal(t, []string{"TEST", "TEST1"}, e.Envs())
}
