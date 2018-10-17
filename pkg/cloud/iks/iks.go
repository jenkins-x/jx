package iks

import (
	gohttp "net/http"

	ibmcloud "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/authentication"
	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/http"
	"github.com/IBM-Cloud/bluemix-go/rest"
	"github.com/IBM-Cloud/bluemix-go/session"
)

type KubernetesServiceAPI interface {
	Zones() Zones
	Regions() Regions
	MachineTypes() MachineTypes
	VLANs() VLANs
	Clusters() Clusters
}

//ContainerService holds the client
type ksService struct {
	*client.Client
}

// No auth session is required for region and zone calls
func NewSessionless(sess *session.Session) (KubernetesServiceAPI, error) {
	config := sess.Config
	if config.HTTPClient == nil {
		config.HTTPClient = http.NewHTTPClient(config)
	}
	if config.Endpoint == nil {
		ep, err := config.EndpointLocator.ContainerEndpoint()
		if err != nil {
			return nil, err
		}
		config.Endpoint = &ep
	}
	return &ksService{
		Client: client.New(config, ibmcloud.ContainerService, nil),
	}, nil
}

func New(sess *session.Session) (KubernetesServiceAPI, error) {
	config := sess.Config.Copy()
	err := config.ValidateConfigForService(ibmcloud.ContainerService)
	if err != nil {
		return nil, err
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.NewHTTPClient(config)
	}
	tokenRefreher, err := authentication.NewIAMAuthRepository(config, &rest.Client{
		DefaultHeader: gohttp.Header{
			"User-Agent": []string{http.UserAgent()},
		},
		HTTPClient: config.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	if config.IAMAccessToken == "" {
		err := authentication.PopulateTokens(tokenRefreher, config)
		if err != nil {
			return nil, err
		}
	}
	if config.Endpoint == nil {
		ep, err := config.EndpointLocator.ContainerEndpoint()
		if err != nil {
			return nil, err
		}
		config.Endpoint = &ep
	}

	return &ksService{
		Client: client.New(config, ibmcloud.ContainerService, tokenRefreher),
	}, nil
}

func (c *ksService) Zones() Zones {
	return newZonesAPI(c.Client)
}
func (c *ksService) Regions() Regions {
	return newRegionsAPI(c.Client)
}
func (c *ksService) MachineTypes() MachineTypes {
	return newMachineTypesAPI(c.Client)
}
func (c *ksService) VLANs() VLANs {
	return newVLANsAPI(c.Client)
}
func (c *ksService) Clusters() Clusters {
	return newClusterAPI(c.Client)
}
