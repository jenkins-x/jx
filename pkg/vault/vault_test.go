// +build unit

package vault_test

import (
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/vault"

	"github.com/jenkins-x/jx/v2/pkg/errorutil"

	"github.com/stretchr/testify/assert"
)

func TestNewExternalVault(t *testing.T) {
	var testConfigs = []struct {
		url                    string
		serviceAccountName     string
		namespace              string
		secretEngineMountPoint string
		kubernetesAuthPath     string
		valid                  bool
		errorCount             int
		errorMessage           string
	}{
		{"", "", "", "", "", false, 1, "URL cannot be empty for an external Vault configuration"},
		{"foo", "", "", "", "", false, 3, "['foo' not a valid URL, external vault service account name cannot be empty, external vault namespace cannot be empty]"},
		{"http://my.vault.com", "", "", "", "", false, 2, "[external vault service account name cannot be empty, external vault namespace cannot be empty]"},
		{"http://my.vault.com", "vault-sa", "", "", "", false, 1, "external vault namespace cannot be empty"},
		{"http://my.vault.com", "vault-sa", "jx", "", "", true, 0, ""},
	}

	for _, testConfig := range testConfigs {
		t.Run(testConfig.url, func(t *testing.T) {
			v, err := vault.NewExternalVault(testConfig.url, testConfig.serviceAccountName, testConfig.namespace, testConfig.secretEngineMountPoint, testConfig.kubernetesAuthPath)
			if testConfig.valid {
				assert.NoError(t, err)
				assert.True(t, v.IsExternal())
				assert.Equal(t, "kubernetes", v.KubernetesAuthPath)
				assert.Equal(t, "secret", v.SecretEngineMountPoint)
			} else {
				assert.Error(t, err)
				if testConfig.errorCount > 1 {
					aggregate := err.(errorutil.Aggregate)
					assert.Len(t, aggregate.Errors(), testConfig.errorCount)
				}
				assert.Equal(t, testConfig.errorMessage, err.Error())
			}
		})
	}
}

func TestNewInternalVault(t *testing.T) {
	var testConfigs = []struct {
		name               string
		serviceAccountName string
		namespace          string
		valid              bool
		errorCount         int
		errorMessage       string
	}{
		{"", "", "", false, 1, "name cannot be empty for an internal Vault configuration"},
		{"foo", "", "", false, 1, "internal vault namespace cannot be empty"},
		{"foo", "vault-sa", "jx", true, 0, ""},
	}

	for _, testConfig := range testConfigs {
		t.Run(testConfig.name, func(t *testing.T) {
			v, err := vault.NewInternalVault(testConfig.name, testConfig.serviceAccountName, testConfig.namespace)
			if testConfig.valid {
				assert.NoError(t, err)
				assert.False(t, v.IsExternal())
				assert.Equal(t, "kubernetes", v.KubernetesAuthPath)
				assert.Equal(t, "secret", v.SecretEngineMountPoint)
			} else {
				assert.Error(t, err)
				if testConfig.errorCount > 1 {
					aggregate := err.(errorutil.Aggregate)
					assert.Len(t, aggregate.Errors(), testConfig.errorCount)
				}
				assert.Equal(t, testConfig.errorMessage, err.Error())
			}
		})
	}
}

func TestFromMap(t *testing.T) {
	var testConfigs = []struct {
		data         map[string]string
		external     bool
		valid        bool
		errorCount   int
		errorMessage string
	}{
		{map[string]string{}, false, false, 2, "[internal vault name cannot be empty, internal vault service account name cannot be empty]"},

		{map[string]string{
			vault.SystemVaultName: "foo",
		}, false, false, 1, "internal vault service account name cannot be empty"},

		{map[string]string{
			vault.SystemVaultName: "foo",
			vault.ServiceAccount:  "bar",
		}, false, true, 0, ""},

		{map[string]string{
			vault.URL:            "http://myvault.acme",
			vault.ServiceAccount: "foo",
			vault.Namespace:      "bar",
		}, true, true, 0, ""},

		{map[string]string{
			vault.URL:             "http://myvault.acme",
			vault.SystemVaultName: "foo",
		}, true, false, 1, "systemVaultName and URL cannot be specified together"},
	}

	for _, testConfig := range testConfigs {
		t.Run(testConfig.data[vault.SystemVaultName], func(t *testing.T) {
			v, err := vault.FromMap(testConfig.data, "foo")
			if testConfig.valid {
				assert.NoError(t, err)
				assert.Equal(t, testConfig.external, v.IsExternal())
				assert.Equal(t, "kubernetes", v.KubernetesAuthPath)
				assert.Equal(t, "secret", v.SecretEngineMountPoint)
			} else {
				assert.Error(t, err)
				if testConfig.errorCount > 1 {
					aggregate := err.(errorutil.Aggregate)
					assert.Len(t, aggregate.Errors(), testConfig.errorCount)
				}
				assert.Equal(t, testConfig.errorMessage, err.Error())
			}
		})
	}
}
