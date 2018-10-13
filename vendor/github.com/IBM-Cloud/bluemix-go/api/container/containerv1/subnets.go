package containerv1

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/client"
)

//Subnet ...
type Subnet struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	VlanID      string           `json:"vlan_id"`
	IPAddresses []string         `json:"ip_addresses"`
	Properties  SubnetProperties `json:"properties"`
}

//SubnetProperties ...
type SubnetProperties struct {
	CIDR              string `json:"cidr"`
	NetworkIdentifier string `json:"network_identifier"`
	Note              string `json:"note"`
	SubnetType        string `json:"subnet_type"`
	DisplayLabel      string `json:"display_label"`
	Gateway           string `json:"gateway"`
}

type UserSubnet struct {
	CIDR   string `json:"cidr" binding:"required" description:"The CIDR of the subnet that will be bound to the cluster. Eg.format: 12.34.56.78/90"`
	VLANID string `json:"vlan_id" binding:"required" description:"The private VLAN where the CIDR exists'"`
}

//Subnets interface
type Subnets interface {
	AddSubnet(clusterName string, subnetID string, target ClusterTargetHeader) error
	List(target ClusterTargetHeader) ([]Subnet, error)
	AddClusterUserSubnet(clusterID string, userSubnet UserSubnet, target ClusterTargetHeader) error
	ListClusterUserSubnets(clusterID string, target ClusterTargetHeader) ([]Vlan, error)
	DeleteClusterUserSubnet(clusterID string, subnetID string, vlanID string, target ClusterTargetHeader) error
}

type subnet struct {
	client *client.Client
}

func newSubnetAPI(c *client.Client) Subnets {
	return &subnet{
		client: c,
	}
}

//GetSubnets ...
func (r *subnet) List(target ClusterTargetHeader) ([]Subnet, error) {
	subnets := []Subnet{}
	_, err := r.client.Get("/v1/subnets", &subnets, target.ToMap())
	if err != nil {
		return nil, err
	}

	return subnets, err
}

//AddSubnetToCluster ...
func (r *subnet) AddSubnet(name string, subnetID string, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/subnets/%s", name, subnetID)
	_, err := r.client.Put(rawURL, nil, nil, target.ToMap())
	return err
}

//AddClusterUserSubnet ...
func (r *subnet) AddClusterUserSubnet(clusterID string, userSubnet UserSubnet, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/usersubnets", clusterID)
	_, err := r.client.Post(rawURL, nil, nil, target.ToMap())
	return err
}

//DeleteClusterUserSubnet ...
func (r *subnet) DeleteClusterUserSubnet(clusterID string, subnetID string, vlanID string, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/usersubnets/%s/vlans/%s", clusterID, subnetID, vlanID)
	_, err := r.client.Delete(rawURL, target.ToMap())
	return err
}

//GetClusterUserSubnet ...
func (r *subnet) ListClusterUserSubnets(clusterID string, target ClusterTargetHeader) ([]Vlan, error) {
	vlans := []Vlan{}
	rawURL := fmt.Sprintf("/v1/clusters/%s/usersubnets", clusterID)
	_, err := r.client.Get(rawURL, &vlans, target.ToMap())
	if err != nil {
		return nil, err
	}

	return vlans, err
}
