package auth

func (s *GenericAuthConfigService) Config() *AuthConfig {
	if s.config == nil {
		s.config = &AuthConfig{}
	}
	return s.config
}

func (s *GenericAuthConfigService) SetConfig(c *AuthConfig) {
	s.config = c
}

// SaveUserAuth saves the given user auth for the server url
func (s *GenericAuthConfigService) SaveUserAuth(url string, userAuth *UserAuth) error {
	config := s.config
	config.SetUserAuth(url, userAuth)
	user := userAuth.Username
	if user != "" {
		config.DefaultUsername = user
	}

	// Set Pipeline user once only.
	if config.PipeLineUsername == "" {
		config.PipeLineUsername = user
		config.PipeLineServer = url
	}

	config.CurrentServer = url
	return s.saver.SaveConfig(s.config)
}

// DeleteServer removes the given server from the configuration
func (s *GenericAuthConfigService) DeleteServer(url string) error {
	s.config.DeleteServer(url)
	return s.saver.SaveConfig(s.config)
}

// LoadConfig loads the configuration from the users JX config directory
func (s *GenericAuthConfigService) LoadConfig() (*AuthConfig, error) {
	return s.saver.LoadConfig()
}

// SaveConfig saves the configuration to disk
func (s *GenericAuthConfigService) SaveConfig() error {
	return s.saver.SaveConfig(s.Config())
}
