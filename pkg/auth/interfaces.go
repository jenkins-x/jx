package auth

// ConfigKind define the type of config
type ConfigKind string

const (
	// LocalConfigKind a local config type which is typically stored locally
	LocalConfigKind ConfigKind = "local"
	// PipelineConfigKind a pipeline config type which is typically used in-cluster by the pipeline
	PipelineConfigKind ConfigKind = "pipeline"
	// AutoConfigKind the config type is auto detected based on the context
	AutoConfigKind ConfigKind = "auto"
)

// ConfigService is a service for handing the config of auth tokens
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/auth ConfigService -o mocks/config_service.go
type ConfigService interface {
	// Config returns the current config
	Config() (*Config, error)
	// SetConfig sets a config
	SetConfig(config *Config)
	// LoadConfig loads the configuration
	LoadConfig() error
	// SaveConfig saves the configuration
	SaveConfig() error
}

// ConfigSaver is an interface that saves an AuthConfig
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/auth ConfigSaver -o mocks/config_saver.go
type ConfigSaver interface {
	// SaveConfig saves the configuration
	SaveConfig(config *Config) error
}

// ConfigLoader is an interface that loads  an AuthConfig
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/auth ConfigLoader -o mocks/config_loader.go
type ConfigLoader interface {
	// LoadConfig loads the configuration from a source
	LoadConfig() (*Config, error)
}

// ConfigLoadSaver is an interface that loads and saves an AuthConfig
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/auth ConfigLoadSaver -o mocks/config_loadsaver.go
type ConfigLoadSaver interface {
	ConfigLoader
	ConfigSaver
}
