package auth

// ConfigService is a service for handing the config of auth tokens
type ConfigService interface {
	Config() *AuthConfig
	SetConfig(c *AuthConfig)
	// LoadConfig loads the configuration from the users JX config directory
	LoadConfig() (*AuthConfig, error)
	//HasConfigFile() (bool, error)
	// SaveConfig saves the configuration
	SaveConfig() error
	// SaveUserAuth saves the given user auth for the server url
	SaveUserAuth(url string, userAuth *UserAuth) error
	// DeleteServer removes the given server from the configuration
	DeleteServer(url string) error
}

// ConfigSaver is an interface that saves an AuthConfig
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/auth ConfigSaver -o mocks/auth_interface.go
type ConfigSaver interface {
	// LoadConfig loads the configuration from the users JX config directory
	LoadConfig() (*AuthConfig, error)
	//HasConfigFile() (bool, error)
	// SaveConfig saves the configuration
	SaveConfig(config *AuthConfig) error
}
