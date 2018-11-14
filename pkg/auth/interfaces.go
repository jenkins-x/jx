package auth

// AuthConfigService is a service for handing the config of auth tokens
type AuthConfigService interface {
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
