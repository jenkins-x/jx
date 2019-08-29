package auth

// NewMemoryAuthConfigService creates a new memory based auth service
func NewMemoryAuthConfigService() ConfigService {
	saver := newMemoryAuthSaver()
	return NewAuthConfigService(saver)
}

func newMemoryAuthSaver() ConfigSaver {
	return &MemoryAuthConfigSaver{}
}

// LoadConfig does nothing
func (m *MemoryAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	return &m.config, nil
}

// SaveConfig updates the config
func (m *MemoryAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	m.config = *config
	return nil
}
