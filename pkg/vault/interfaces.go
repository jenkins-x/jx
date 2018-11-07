package vault

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/common"
	"net/url"
)

// Vaulter is an interface for creating new vault clients
// We _don't_ want this to just be a mirror of the official api.Client object, as for most of the time you would just
// want to use the underlying client.
type Vaulter interface {
	// Config gets the config required for configuring the official Vault CLI
	Config() (vaultUrl url.URL, vaultToken string, err error)
	// Secrets lists the secrets stored in the vault. Beta - subject to change
	Secrets() ([]string, error)
}

// VaultSelector is an interface for selecting a vault from the installed ones on the platform
// It should pick the most logical one, or give the user a way of picking a vault if there are multiple installed
type VaultSelector interface {
	GetVault(name string, namespace string) (*Vault, error)
}

type VaultOptions interface {
	common.OptionsInterface
	VaultName() string
	VaultNamespace() string
}
