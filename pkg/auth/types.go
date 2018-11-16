package auth

import "github.com/jenkins-x/jx/pkg/vault"

const (
	DefaultWritePermissions = 0760
)

type AuthServer struct {
	URL   string
	Users []*UserAuth
	Name  string
	Kind  string

	CurrentUser string
}

type UserAuth struct {
	Username    string
	ApiToken    string
	BearerToken string
	Password    string `yaml:"password,omitempty"`
}

type AuthConfig struct {
	Servers []*AuthServer

	DefaultUsername  string
	CurrentServer    string
	PipeLineUsername string
	PipeLineServer   string
}

// GenericAuthConfigService implements the generic features of the ConfigService because we don't have superclasses
type GenericAuthConfigService struct {
	config *AuthConfig
	saver  ConfigSaver
}

// FileBasedAuthConfigSaver is a ConfigSaver that saves its config to the local filesystem
type FileBasedAuthConfigSaver struct {
	FileName string
}

// VaultBasedAuthConfigService is an ConfigService that stores its secret data in a Vault
type VaultBasedAuthConfigService struct {
	vaulter vault.Vaulter
}
