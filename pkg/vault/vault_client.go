package vault

import (
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type VaultClient interface {
	// Write writes a named secret to the vault
	Write(secretName string, data map[string]interface{}) (map[string]interface{}, error)

	// WriteObject writes a generic named object to the vault. The secret _must_ be serializable to JSON
	WriteObject(secretName string, secret interface{}) (map[string]interface{}, error)

	// WriteYaml writes a yaml object to a named secret
	WriteYaml(secretName string, yamlstring string) (map[string]interface{}, error)

	// Read reads a named secret from the vault
	Read(secretName string) (map[string]interface{}, error)
}

// VaultClient is a hand wrapper around the official Vault API to save shit in the way we want
type VaultClientImpl struct {
	client *api.Client
}

// NewVaultClient creates a new Vault Client wrapping the api.client
func NewVaultClient(client *api.Client) VaultClient {
	return &VaultClientImpl{client: client}
}

// Write writes a named secret to the vault with the data provided. Data can be a generic map of stuff, but at all points
// in the map, keys _must_ be strings (not bool, int or even interface{}) otherwise you'll get an error
func (v *VaultClientImpl) Write(secretName string, data map[string]interface{}) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Write(secretPath(secretName), data)
	if secret != nil {
		return secret.Data, err
	}
	return nil, err
}

// WriteObject writes a generic named object to the vault. The secret _must_ be serializable to JSON
func (v *VaultClientImpl) WriteObject(secretName string, secret interface{}) (map[string]interface{}, error) {
	// Convert the secret into a saveable map[string]interface{} format
	m, err := util.ToMapStringInterfaceFromStruct(&secret)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not serialize secret '%s' object for saving to vault", secretName)
	}
	return v.Write(secretName, m)
}

// WriteYaml writes a yaml object to a named secret
func (v *VaultClientImpl) WriteYaml(secretName string, y string) (map[string]interface{}, error) {
	// Unmarshal to a generic map
	m := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(y), &m)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not unmarshal YAML %v", y)
	}

	// We can't just call v.client.save on this. Although it is a map[string]interface{}, a sub-item in the may _may_
	// be a map[interface{}]interface rather than map[string]interface{}. This will cause the vault Write action to fail
	// Instead we need to marshall to a struct and back
	out := util.ConvertAllMapKeysToString(m)
	return v.Write(secretName, out.(map[string]interface{}))
}

// Read reads a named secret to the vault
func (v *VaultClientImpl) Read(secretName string) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Read(secretPath(secretName))
	if secret != nil {
		return secret.Data, err
	}
	return nil, err
}

// secretPath generates a secret path from the secret name for storing in vault
// this just makes sure it gets stored under /secret
func secretPath(secretName string) string {
	return "secret/" + secretName
}
