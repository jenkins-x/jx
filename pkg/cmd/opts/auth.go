package opts

import (
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/auth"
)

// CreateGitConfigService creates git auth config service
func (o *CommonOptions) CreateGitConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateGitConfigService(auth.AutoConfigKind)
}

// CurrentGitServerAuth returns the current git server auth detected automatically based on the
// context (pipeline|local)
func (o *CommonOptions) CurrentGitServerAuth() (auth.Server, error) {
	server := auth.Server{}
	cs, err := o.CreateGitConfigService()
	if err != nil {
		return server, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return server, err
	}
	return cfg.GetCurrentServer()
}

// CurrentGitServerAuthForPipeline returns the current git server auth config for pipeline
func (o *CommonOptions) CurrentGitServerAuthForPipeline() (auth.Server, error) {
	server := auth.Server{}
	if o.factory == nil {
		return server, errors.New("command factory is not initialized")
	}
	cs, err := o.factory.CreateGitConfigService(auth.PipelineConfigKind)
	if err != nil {
		return server, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return server, err
	}
	return cfg.GetCurrentServer()
}

// CreateAuthConfigService creates a new chat auth service
func (o *CommonOptions) CreateChatConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateChatConfigService(auth.AutoConfigKind)
}

// AddonConfigService creates the addon auth config service
func (o *CommonOptions) CreateAddonConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateAddonConfigService(auth.AutoConfigKind)
}

// JenkinsConfigService creates the jenkins auth config service
func (o *CommonOptions) CreateJenkinsConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateJenkinsConfigService(auth.AutoConfigKind)
}

// ChartmuseumConfigService creates the chart museum auth config service
func (o *CommonOptions) CreateChartmuseumConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateChartmuseumConfigService(auth.AutoConfigKind)
}

// GetServerConfig returns the server configuration from server flgas values
func (o *CommonOptions) GetServerConfig(service auth.ConfigService, serverFlags *ServerFlags) (auth.Server, error) {
	cfg, err := service.Config()
	if err != nil {
		return auth.Server{}, err
	}
	server, err := cfg.GetServer(serverFlags.ServerURL)
	if err == nil {
		return server, nil
	}
	return cfg.GetServerByName(serverFlags.ServerName)
}
