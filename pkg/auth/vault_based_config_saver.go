package auth

import (
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/util"
)

// VaultBasedAuthConfigSaver is a ConfigSaver that saves configs to Vault
type VaultBasedAuthConfigSaver struct {
	vaultClient *api.Client
	secretName  string
}

// LoadConfig loads the config from the vault
func (v *VaultBasedAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	data, err := v.vaultClient.Logical().Read(secretPath(v.secretName))
	if err != nil {
		return nil, err
	}
	config := AuthConfig{}

	err = util.ToStructFromMapStringInterface(data.Data, &config)
	return &config, err
}

// SaveConfig saves the config to the vault
func (v *VaultBasedAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	// Marshall the AuthConfig to a generic map to save in vault (as that's what vault takes)
	m, err := util.ToMapStringInterfaceFromStruct(&config)
	if err == nil {
		v.vaultClient.Logical().Write(secretPath(v.secretName), m)
	}
	return err
}

// NewVaultBasedAuthConfigSaver creates a ConfigSaver that saves the Configs under a specified secretname in a vault
func NewVaultBasedAuthConfigSaver(secretName string, vaultClient *api.Client) VaultBasedAuthConfigSaver {
	return VaultBasedAuthConfigSaver{
		secretName:  secretName,
		vaultClient: vaultClient,
	}
}

// secretPath generates a secret path from the secret name for storing in vault
// this just makes sure it gets stored under /secret
func secretPath(secretName string) string {
	return "secret/" + secretName
}
