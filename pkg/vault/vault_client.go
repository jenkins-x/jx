package vault

import "github.com/hashicorp/vault/api"

type VaultClient interface {
	// Write writes a named secret to the vault
	Write(secretName string, data map[string]interface{}) (map[string]interface{}, error)

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
