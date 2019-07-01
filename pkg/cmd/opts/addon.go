package opts

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/pkg/errors"
)

// GetAddonServer returns the server configuration for a given service URL
func (o *CommonOptions) GetAddonServer(serverURL string) (auth.Server, error) {
	server := auth.Server{}
	if serverURL == "" {
		return server, errors.New("cannot find server configuration for an empty addon URL")
	}
	cs, err := o.CreateAddonConfigService()
	if err != nil {
		return server, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return server, err
	}
	return cfg.GetServer(serverURL)
}

// GetAddonServerByKind returns the server for a given addon kind
func (o *CommonOptions) GetAddonAuthByKind(kind string) (auth.Server, error) {
	server := auth.Server{}
	if kind == "" {
		return server, errors.New("connot find server configuration for an empty addon kind")
	}
	cs, err := o.CreateAddonConfigService()
	if err != nil {
		return server, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return server, err
	}
	return cfg.GetServerByKind(kind)
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
