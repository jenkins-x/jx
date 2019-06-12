package opts

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube/services"
)

// GetAddonAuth returns the server and user auth for the given addon service URL
func (o *CommonOptions) GetAddonAuth(serviceURL string) (*auth.AuthServer, *auth.UserAuth, error) {
	if serviceURL == "" {
		return nil, nil, nil
	}
	authConfigSvc, err := o.CreateAddonAuthConfigService()
	if err != nil {
		return nil, nil, err
	}
	config := authConfigSvc.Config()

	server := config.GetOrCreateServer(serviceURL)
	userAuth, err := config.PickServerUserAuth(server, "user to access the addon service at "+serviceURL, o.BatchMode, "", o.In, o.Out, o.Err)
	return server, userAuth, err
}

// GetAddonAuth returns the server and user auth for the given addon service URL. Returns null values if there is no server
func (o *CommonOptions) GetAddonAuthByKind(kind, serverURL string) (*auth.AuthServer, *auth.UserAuth, error) {
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
	userAuth, err := config.PickServerUserAuth(server, message, true, "", o.In, o.Out, o.Err)
	return server, userAuth, err
}

// EnsureAddonServiceAvailable ensures that the given service name is available for addon
func (o *CommonOptions) EnsureAddonServiceAvailable(serviceName string) (string, error) {
	present, err := services.IsServicePresent(o.kubeClient, serviceName, o.currentNamespace)
	if err != nil {
		return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
	}
	if present {
		url, err := services.GetServiceURLFromName(o.kubeClient, serviceName, o.currentNamespace)
		if err != nil {
			return "", fmt.Errorf("no %s provider service found, are you in your teams dev environment?  Type `jx ns` to switch.", serviceName)
		}
		return url, nil
	}

	// todo ask if user wants to install addon?
	return "", nil
}
