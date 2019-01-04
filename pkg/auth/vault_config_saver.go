package auth

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/vault"
)

// LoadConfig loads the config from the vault
func (v *VaultAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	data, err := v.vaultClient.Read(vault.AuthSecretPath(v.secretName))
	if err != nil {
		return nil, err
	}
	config := AuthConfig{}

	if data != nil {
		err = util.ToStructFromMapStringInterface(data, &config)
	}
	return &config, err
}

// SaveConfig saves the config to the vault
func (v *VaultAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	// Marshall the AuthConfig to a generic map to save in vault (as that's what vault takes)
	m, err := util.ToMapStringInterfaceFromStruct(&config)
	if err == nil {
		_, err = v.vaultClient.Write(vault.AuthSecretPath(v.secretName), m)
	}
	return err
}

// NewVaultAuthConfigService creates a new ConfigService that saves it config to a Vault
func NewVaultAuthConfigService(secretName string, vaultClient vault.Client) ConfigService {
	saver := newVaultAuthConfigSaver(secretName, vaultClient)
	return NewAuthConfigService(&saver)
}

// newVaultAuthConfigSaver creates a ConfigSaver that saves the Configs under a specified secretname in a vault
func newVaultAuthConfigSaver(secretName string, vaultClient vault.Client) VaultAuthConfigSaver {
	return VaultAuthConfigSaver{
		secretName:  secretName,
		vaultClient: vaultClient,
	}
}
