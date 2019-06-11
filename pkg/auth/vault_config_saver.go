package auth

import (
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"

	"sigs.k8s.io/yaml"
)

// vaultAuthConfigSaver is a ConfigSaver that saves configs to Vault
type vaultAuthConfigSaver struct {
	vaultClient vault.Client
	secretName  string
}

// LoadConfig loads the config from the vault
func (v *vaultAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	data, err := v.vaultClient.ReadYaml(vault.AuthSecretPath(v.secretName))
	if err != nil {
		return nil, errors.Wrapf(err, "loading the auth config %q from vault", v.secretName)
	}

	var config AuthConfig
	if data == "" {
		return &config, nil
	}

	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		return nil, errors.Wrapf(err, "unmarshalling auth config %q", v.secretName)
	}
	return &config, nil
}

// SaveConfig saves the config to the vault
func (v *vaultAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshaling auth config")
	}
	if _, err := v.vaultClient.WriteYaml(vault.AuthSecretPath(v.secretName), string(data)); err != nil {
		return errors.Wrapf(err, "saving auth config %q in vault", v.secretName)
	}
	return nil
}

// NewVaultAuthConfigService creates a new ConfigService that saves it config to a Vault
func NewVaultAuthConfigService(secretName string, vaultClient vault.Client) ConfigService {
	saver := newVaultAuthConfigSaver(secretName, vaultClient)
	return NewAuthConfigService(&saver)
}

// newVaultAuthConfigSaver creates a ConfigSaver that saves the Configs under a specified secretname in a vault
func newVaultAuthConfigSaver(secretName string, vaultClient vault.Client) vaultAuthConfigSaver {
	return vaultAuthConfigSaver{
		secretName:  secretName,
		vaultClient: vaultClient,
	}
}
