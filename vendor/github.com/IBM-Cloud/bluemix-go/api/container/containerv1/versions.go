package containerv1

import (
	"github.com/IBM-Cloud/bluemix-go/client"
)

//KubeVersion ...
type KubeVersion struct {
	Major   int
	Minor   int
	Patch   int
	Default bool
}

//KubeVersions interface
type KubeVersions interface {
	List(target ClusterTargetHeader) ([]KubeVersion, error)
}

type version struct {
	client *client.Client
}

func newKubeVersionAPI(c *client.Client) KubeVersions {
	return &version{
		client: c,
	}
}

//List ...
func (v *version) List(target ClusterTargetHeader) ([]KubeVersion, error) {
	versions := []KubeVersion{}
	_, err := v.client.Get("/v1/kube-versions", &versions, target.ToMap())
	if err != nil {
		return nil, err
	}
	return versions, err
}
