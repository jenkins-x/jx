package vault

import (
	"fmt"
	"github.com/hashicorp/vault/api"
)

// GetSecrets returns a list of secrets from Vault
// It's a bit hacky at the moment
func GetSecrets(client *api.Client) ([]string, error) {
	secrets, err := client.Logical().List("secret")
	if err != nil {
		return nil, err
	}

	out := make([]string, 0)
	for key, value := range secrets.Data {
		_ = key
		_ = value
		out = append(out, fmt.Sprintf("%v", value))
		out = append(out, fmt.Sprintf("%T", value))
	}

	return out, nil
}
