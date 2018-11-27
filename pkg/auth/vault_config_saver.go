package auth

import (
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/util"
)

// VaultAuthConfigSaver is a ConfigSaver that saves configs to Vault
type VaultAuthConfigSaver struct {
	vaultClient *api.Client
	secretName  string
}

// LoadConfig loads the config from the vault
func (v *VaultAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	data, err := v.vaultClient.Logical().Read(secretPath(v.secretName))
	if err != nil {
		return nil, err
	}
	config := AuthConfig{}

	if data != nil {
		err = util.ToStructFromMapStringInterface(data.Data, &config)
	}
	return &config, err
}

// SaveConfig saves the config to the vault
func (v *VaultAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	// Marshall the AuthConfig to a generic map to save in vault (as that's what vault takes)
	m, err := util.ToMapStringInterfaceFromStruct(&config)
	if err == nil {
		v.vaultClient.Logical().Write(secretPath(v.secretName), m)
	}
	return err
}

// NewVaultAuthConfigService creates a new ConfigService that saves it config to a Vault
func NewVaultAuthConfigService(secretName string, vaultClient *api.Client) ConfigService {
	saver := newVaultAuthConfigSaver(secretName, vaultClient)
	return NewAuthConfigService(&saver)
}

// newVaultAuthConfigSaver creates a ConfigSaver that saves the Configs under a specified secretname in a vault
func newVaultAuthConfigSaver(secretName string, vaultClient *api.Client) VaultAuthConfigSaver {
	return VaultAuthConfigSaver{
		secretName:  secretName,
		vaultClient: vaultClient,
	}
}

// secretPath generates a secret path from the secret name for storing in vault
// this just makes sure it gets stored under /secret
func secretPath(secretName string) string {
	return "secret/" + secretName
}
