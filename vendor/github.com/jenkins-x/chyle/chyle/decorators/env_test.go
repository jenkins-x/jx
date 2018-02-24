package decorators

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	err := os.Setenv("TESTENVDECORATOR", "this is a test")

	assert.NoError(t, err)

	envs := map[string]struct {
		DESTKEY string
		VARNAME string
	}{
		"WHATEVER": {
			"envTesting",
			"TESTENVDECORATOR",
		},
	}

	metadatas := map[string]interface{}{}

	e := newEnvs(envs)
	m, err := e[0].Decorate(&metadatas)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"envTesting": "this is a test"}, *m)
}
