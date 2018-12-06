package vault

import (
	"net/url"

	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Client is an interface for interacting with Vault
type Client interface {
	// Write writes a named secret to the vault
	Write(secretName string, data map[string]interface{}) (map[string]interface{}, error)

	// WriteObject writes a generic named object to the vault. The secret _must_ be serializable to JSON
	WriteObject(secretName string, secret interface{}) (map[string]interface{}, error)

	// WriteSecrets writes a generic Map of secrets to vault under a specific path
	WriteSecrets(path string, secretsToSave map[string]interface{}) error

	// WriteYaml writes a yaml object to a named secret
	WriteYaml(secretName string, yamlstring string) (map[string]interface{}, error)

	// List lists the secrets under the specified path
	List(path string) ([]string, error)

	// Read reads a named secret from the vault
	Read(secretName string) (map[string]interface{}, error)

	// Config gets the config required for configuring the official Vault CLI
	Config() (vaultURL url.URL, vaultToken string, err error)
}

// client is a hand wrapper around the official Vault API
type client struct {
	client *api.Client
}

// NewVaultClient creates a new Vault Client wrapping the api.client
func NewVaultClient(apiclient *api.Client) Client {
	return &client{client: apiclient}
}

// Write writes a named secret to the vault with the data provided. Data can be a generic map of stuff, but at all points
// in the map, keys _must_ be strings (not bool, int or even interface{}) otherwise you'll get an error
func (v *client) Write(secretName string, data map[string]interface{}) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Write(secretPath(secretName), data)
	if secret != nil {
		return secret.Data, err
	}
	return nil, err
}

// WriteObject writes a generic named object to the vault. The secret _must_ be serializable to JSON
func (v *client) WriteObject(secretName string, secret interface{}) (map[string]interface{}, error) {
	// Convert the secret into a saveable map[string]interface{} format
	m, err := util.ToMapStringInterfaceFromStruct(&secret)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not serialize secret '%s' object for saving to vault", secretName)
	}
	return v.Write(secretName, m)
}

// WriteSecrets writes a generic Map of secrets to vault under a specific path
func (v *client) WriteSecrets(path string, secretsToSave map[string]interface{}) error {
	var err error
	for secretName, secret := range secretsToSave {
		secretName = secretName + path
		switch secret.(type) {
		case []byte:
			// secret is a plain byte array. We shouldn't be doing this. Legacy. We should be saving properly typed objects
			_, err = v.WriteYaml(secretName, string(secret.([]byte)[:]))
		case string:
			// secret is a string. We shouldn't be doing this. Legacy. We should be saving properly typed objects
			_, err = v.WriteYaml(secretName, secret.(string))
		default:
			// secret is an interface. This is what we should be doing
			_, err = v.WriteObject(secretName, secret)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteYaml writes a yaml object to a named secret
func (v *client) WriteYaml(secretName string, y string) (map[string]interface{}, error) {
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

// List lists the secrets under a given path
func (v *client) List(path string) ([]string, error) {
	secrets, err := v.client.Logical().List(secretPath(path))
	if err != nil {
		return nil, err
	}

	secretNames := make([]string, 0)
	if secrets == nil {
		return secretNames, nil
	}
	for _, s := range secrets.Data["keys"].([]interface{}) {
		if orig, ok := s.(string); ok {
			secretNames = append(secretNames, orig)
		}
	}

	return secretNames, nil
}

// Read reads a named secret to the vault
func (v *client) Read(secretName string) (map[string]interface{}, error) {
	secret, err := v.client.Logical().Read(secretPath(secretName))
	if secret != nil {
		return secret.Data, err
	}
	return nil, err
}

// Config retruns the current vault address and api token
func (v *client) Config() (vaultUrl url.URL, vaultToken string, err error) {
	parsed, err := url.Parse(v.client.Address())
	return *parsed, v.client.Token(), err
}

// secretPath generates a secret path from the secret name for storing in vault
// this just makes sure it gets stored under /secret
func secretPath(secretName string) string {
	return "secret/" + secretName
}
