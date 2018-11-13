package vault

import (
	"github.com/hashicorp/vault/api"
	"net/url"
)

type JxVaulter struct {
	client *api.Client
}

func newJxVaulter(client *api.Client) Vaulter {
	return &JxVaulter{client: client}
}

func NewVaulter(o VaultOptions) (Vaulter, error) {
	clientFactory, err := NewVaultClientFactory(o)
	client, err := clientFactory.NewVaultClient(o.VaultName(), o.VaultNamespace())
	return newJxVaulter(client), err
}

func (v *JxVaulter) Config() (vaultUrl url.URL, vaultToken string, err error) {
	parsed, err := url.Parse(v.client.Address())
	return *parsed, v.client.Token(), err
}

func (v *JxVaulter) Secrets() ([]string, error) {
	// Change this when we decide what kind of schema/pattern we should use for storing secrets (which will be better
	// understood when we start to store secrets)
	secrets, err := v.client.Logical().List("secret")
	if err != nil {
		return nil, err
	}

	out := make([]string, 0)
	for _, s := range secrets.Data["keys"].([]interface{}) {
		if orig, ok := s.(string); ok {
			out = append(out, orig)
		}
	}

	return out, nil
}
