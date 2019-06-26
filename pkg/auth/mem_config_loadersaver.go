package auth

type memConfigLoadSaver struct {
	config Config
}

// NewMemConfigService creates a new memory config service
func NewMemConfigService(config Config) (ConfigService, error) {
	mls := &memConfigLoadSaver{
		config: config,
	}
	return NewConfigService(mls, mls), nil
}

// LoadConfig loads the configuration
func (m *memConfigLoadSaver) LoadConfig() (*Config, error) {
	return &m.config, nil
}

// SaveConfig saves the configuration
func (m *memConfigLoadSaver) SaveConfig(config *Config) error {
	m.config = *config
	return nil
}
