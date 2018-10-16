package containerv1

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/client"
)

//WebHook is the web hook
type WebHook struct {
	Level string
	Type  string
	URL   string
}

//Webhooks interface
type Webhooks interface {
	List(clusterName string, target ClusterTargetHeader) ([]WebHook, error)
	Add(clusterName string, params WebHook, target ClusterTargetHeader) error
}

type webhook struct {
	client *client.Client
}

func newWebhookAPI(c *client.Client) Webhooks {
	return &webhook{
		client: c,
	}
}

//List ...
func (r *webhook) List(name string, target ClusterTargetHeader) ([]WebHook, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s/webhooks", name)
	webhooks := []WebHook{}
	_, err := r.client.Get(rawURL, &webhooks, target.ToMap())
	if err != nil {
		return nil, err
	}

	return webhooks, err
}

//Add ...
func (r *webhook) Add(name string, params WebHook, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/webhooks", name)
	_, err := r.client.Post(rawURL, params, nil, target.ToMap())
	return err
}
