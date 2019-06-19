package secreturl

// Client is a simple interface for Vault-like things such as `vault.Client` or a file system we can use to
// access secret files and values in helm.
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/vault Client -o mocks/vault_client.go
type Client interface {
	// Read reads a named secret from the vault
	Read(secretName string) (map[string]interface{}, error)

	// ReadObject reads a generic named object from vault.
	// The secret _must_ be serializable to JSON.
	ReadObject(secretName string, secret interface{}) error

	// WriteObject writes a generic named object to the vault.
	// The secret _must_ be serializable to JSON.
	WriteObject(secretName string, secret interface{}) (map[string]interface{}, error)

	// ReplaceURIs will replace any vault: URIs in a string (or whatever URL scheme the secret URL client supports
	ReplaceURIs(text string) (string, error)
}
