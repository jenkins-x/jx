package config

import (
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

func TestAdminSecrets(t *testing.T) {

	testFile, err := ioutil.ReadFile("admin_secrets_test.yaml")
	assert.NoError(t, err)

	service := AdminSecretsService{}
	service.Flags.DefaultAdminPassword = "mysecret"
	err = service.NewAdminSecretsConfig()
	assert.NoError(t, err)

	s, err := service.Secrets.String()
	assert.NoError(t, err)

	assert.Equal(t, string(testFile), s, "expected admin secret values do not match")
}
