package auth

import "github.com/pkg/errors"

// configService is responsible to load and save the Config
type configService struct {
	config *Config
	saver  ConfigSaver
	loader ConfigLoader
}

// Config gets the AuthConfig from the source
func (s *configService) Config() (*Config, error) {
	if s.config == nil {
		if err := s.LoadConfig(); err != nil {
			return nil, err
		}
	}
	return s.config, nil
}

// SetConfig sets an AuthConfig object
func (s *configService) SetConfig(c *Config) {
	s.config = c
}

// LoadConfig loads the configuration from the source
func (s *configService) LoadConfig() error {
	if s.loader == nil {
		return errors.New("no config loader is available")
	}
	var err error
	config, err := s.loader.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "loading the config")
	}
	s.config = config
	return nil
}

// SaveConfig saves the configuration to a source
func (s *configService) SaveConfig() error {
	// Skip saving the configuration if no saver is set
	if s.saver == nil {
		return errors.New("no config saver is available")
	}
	config, err := s.Config()
	if err != nil {
		return err
	}
	return s.saver.SaveConfig(config)
}

// newConfigService generates a ConfigService with a custom saver and loader
func newConfigService(saver ConfigSaver, loader ConfigLoader) ConfigService {
	return &configService{saver: saver, loader: loader}
}
