package containerv1

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/client"
)

//Vlan ...
type DCVlan struct {
	ID         string           `json:"id"`
	Properties DCVlanProperties `json:"properties"`
	Type       string           `json:"type"`
}

//VlanProperties ...
type DCVlanProperties struct {
	LocalDiskStorageCapability string `json:"local_disk_storage_capability"`
	Location                   string `json:"location"`
	Name                       string `json:"name"`
	Note                       string `json:"note"`
	PrimaryRouter              string `json:"primary_router"`
	SANStorageCapability       string `json:"san_storage_capability"`
	VlanNumber                 string `json:"vlan_number"`
	VlanType                   string `json:"vlan_type"`
}

//Subnets interface
type Vlans interface {
	List(datacenter string, target ClusterTargetHeader) ([]DCVlan, error)
}

type vlan struct {
	client *client.Client
}

func newVlanAPI(c *client.Client) Vlans {
	return &vlan{
		client: c,
	}
}

//GetVlans ...
func (r *vlan) List(datacenter string, target ClusterTargetHeader) ([]DCVlan, error) {
	vlans := []DCVlan{}
	rawURL := fmt.Sprintf("/v1/datacenters/%s/vlans", datacenter)
	_, err := r.client.Get(rawURL, &vlans, target.ToMap())
	if err != nil {
		return nil, err
	}

	return vlans, err
}
