package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube"
)

// getAddonAuth returns the server and user auth for the given addon service URL
func (o *CommonOptions) getAddonAuth(serviceURL string) (*auth.AuthServer, *auth.UserAuth, error) {
	if serviceURL == "" {
		return nil, nil, nil
	}
	authConfigSvc, err := o.CreateAddonAuthConfigService()
	if err != nil {
		return nil, nil, err
	}
	config := authConfigSvc.Config()

	server := config.GetOrCreateServer(serviceURL)
	userAuth, err := config.PickServerUserAuth(server, "user to access the addon service at "+serviceURL, o.BatchMode)
	return server, userAuth, err
}

// getAddonAuth returns the server and user auth for the given addon service URL. Returns null values if there is no server
func (o *CommonOptions) getAddonAuthByKind(kind, serverURL string) (*auth.AuthServer, *auth.UserAuth, error) {
	authConfigSvc, err := o.CreateAddonAuthConfigService()
	if err != nil {
		return nil, nil, err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	for _, s := range config.Servers {
		if s.Kind == kind && s.URL == serverURL {
			server = s
		}
	}
	if server == nil {
		// TODO lets try find the service in the current namespace using a naming convention?
		return nil, nil, fmt.Errorf("no server found for kind %s", kind)
	}
	message := "user to access the " + kind + " addon service at " + server.URL
	userAuth, err := config.PickServerUserAuth(server, message, true)
	return server, userAuth, err
}

func (o *CommonOptions) CreateAddonAuthConfigService() (auth.AuthConfigService, error) {
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindAddon, "")
	if err != nil {
		o.warnf("The current user cannot query pipeline addon secrets: %s", err)
	}
	return o.Factory.CreateAddonAuthConfigService(secrets)
}
