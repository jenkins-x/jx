package auth

import (
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"

	"sigs.k8s.io/yaml"
)

// vaultConfigLoadSaver loads or store config into vault
type vaultConfigLoadSaver struct {
	vaultClient vault.Client
	secretName  string
}

// LoadConfig loads the config from the vault
func (v *vaultConfigLoadSaver) LoadConfig() (*Config, error) {
	data, err := v.vaultClient.ReadYaml(vault.AuthSecretPath(v.secretName))
	if err != nil {
		return nil, errors.Wrapf(err, "loading the auth config %q from vault", v.secretName)
	}

	var config Config
	if data == "" {
		return &config, nil
	}

	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		return nil, errors.Wrapf(err, "unmarshalling auth config %q", v.secretName)
	}
	return &config, nil
}

// SaveConfig saves the config to the vault
func (v *vaultConfigLoadSaver) SaveConfig(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshaling auth config")
	}
	if _, err := v.vaultClient.WriteYaml(vault.AuthSecretPath(v.secretName), string(data)); err != nil {
		return errors.Wrapf(err, "saving auth config %q in vault", v.secretName)
	}
	return nil
}

// NewVaultConfigService creates a new ConfigService that saves it config to a Vault
func NewVaultConfigService(secretName string, vaultClient vault.Client) ConfigService {
	vls := newVaultConfigLoadSaver(secretName, vaultClient)
	return NewConfigService(&vls, &vls)
}

// newVaultConfigSaver creates a ConfigSaver that saves the Configs under a specified secretname in a vault
func newVaultConfigLoadSaver(secretName string, vaultClient vault.Client) vaultConfigLoadSaver {
	return vaultConfigLoadSaver{
		secretName:  secretName,
		vaultClient: vaultClient,
	}
}
