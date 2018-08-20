package config

import (
	"io/ioutil"
	"testing"

	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestAdminSecrets(t *testing.T) {
	t.Parallel()

	testFile, err := ioutil.ReadFile("admin_secrets_test.yaml")
	assert.NoError(t, err)

	service := AdminSecretsService{}
	service.Flags.DefaultAdminPassword = "mysecret"
	err = service.NewAdminSecretsConfig()
	assert.NoError(t, err)

	s, err := service.Secrets.String()
	tests.Debugf("%s", s)
	assert.NoError(t, err)

	assert.Equal(t, string(testFile), s, "expected admin secret values do not match")
}
