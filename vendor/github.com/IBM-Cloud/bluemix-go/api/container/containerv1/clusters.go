package containerv1

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/trace"
)

//ClusterInfo ...
type ClusterInfo struct {
	CreatedDate              string   `json:"createdDate"`
	DataCenter               string   `json:"dataCenter"`
	ID                       string   `json:"id"`
	IngressHostname          string   `json:"ingressHostname"`
	IngressSecretName        string   `json:"ingressSecretName"`
	Location                 string   `json:"location"`
	MasterKubeVersion        string   `json:"masterKubeVersion"`
	ModifiedDate             string   `json:"modifiedDate"`
	Name                     string   `json:"name"`
	Region                   string   `json:"region"`
	ResourceGroupID          string   `json:"resourceGroup"`
	ServerURL                string   `json:"serverURL"`
	State                    string   `json:"state"`
	OrgID                    string   `json:"logOrg"`
	OrgName                  string   `json:"logOrgName"`
	SpaceID                  string   `json:"logSpace"`
	SpaceName                string   `json:"logSpaceName"`
	IsPaid                   bool     `json:"isPaid"`
	IsTrusted                bool     `json:"isTrusted"`
	WorkerCount              int      `json:"workerCount"`
	Vlans                    []Vlan   `json:"vlans"`
	Addons                   []Addon  `json:"addons"`
	OwnerEmail               string   `json:"ownerEmail"`
	APIUser                  string   `json:"apiUser"`
	MonitoringURL            string   `json:"monitoringURL"`
	DisableAutoUpdate        bool     `json:"disableAutoUpdate"`
	EtcdPort                 string   `json:"etcdPort"`
	MasterStatus             string   `json:"masterStatus"`
	MasterStatusModifiedDate string   `json:"masterStatusModifiedDate"`
	KeyProtectEnabled        bool     `json:"keyProtectEnabled"`
	WorkerZones              []string `json:"workerZones"`
}

type ClusterUpdateParam struct {
	Action  string `json:"action"`
	Force   bool   `json:"force"`
	Version string `json:"version"`
}

type Vlan struct {
	ID      string `json:"id"`
	Subnets []struct {
		Cidr     string   `json:"cidr"`
		ID       string   `json:"id"`
		Ips      []string `json:"ips"`
		IsByOIP  bool     `json:"is_byoip"`
		IsPublic bool     `json:"is_public"`
	}
	Zone   string `json:"zone"`
	Region string `json:"region"`
}

type Addon struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

//ClusterCreateResponse ...
type ClusterCreateResponse struct {
	ID string
}

//ClusterTargetHeader ...
type ClusterTargetHeader struct {
	OrgID         string
	SpaceID       string
	AccountID     string
	Region        string
	ResourceGroup string
}

const (
	orgIDHeader         = "X-Auth-Resource-Org"
	spaceIDHeader       = "X-Auth-Resource-Space"
	accountIDHeader     = "X-Auth-Resource-Account"
	slUserNameHeader    = "X-Auth-Softlayer-Username"
	slAPIKeyHeader      = "X-Auth-Softlayer-APIKey"
	regionHeader        = "X-Region"
	resourceGroupHeader = "X-Auth-Resource-Group"
)

//ToMap ...
func (c ClusterTargetHeader) ToMap() map[string]string {
	m := make(map[string]string, 3)
	m[orgIDHeader] = c.OrgID
	m[spaceIDHeader] = c.SpaceID
	m[accountIDHeader] = c.AccountID
	m[regionHeader] = c.Region
	m[resourceGroupHeader] = c.ResourceGroup
	return m
}

//ClusterSoftlayerHeader ...
type ClusterSoftlayerHeader struct {
	SoftLayerUsername string
	SoftLayerAPIKey   string
}

//ToMap ...
func (c ClusterSoftlayerHeader) ToMap() map[string]string {
	m := make(map[string]string, 2)
	m[slAPIKeyHeader] = c.SoftLayerAPIKey
	m[slUserNameHeader] = c.SoftLayerUsername
	return m
}

//ClusterCreateRequest ...
type ClusterCreateRequest struct {
	Billing        string `json:"billing,omitempty"`
	Datacenter     string `json:"dataCenter" description:"The worker's data center"`
	Isolation      string `json:"isolation" description:"Can be 'public' or 'private'"`
	MachineType    string `json:"machineType" description:"The worker's machine type"`
	Name           string `json:"name" binding:"required" description:"The cluster's name"`
	PrivateVlan    string `json:"privateVlan" description:"The worker's private vlan"`
	PublicVlan     string `json:"publicVlan" description:"The worker's public vlan"`
	WorkerNum      int    `json:"workerNum,omitempty" binding:"required" description:"The number of workers"`
	NoSubnet       bool   `json:"noSubnet" description:"Indicate whether portable subnet should be ordered for user"`
	MasterVersion  string `json:"masterVersion,omitempty" description:"Desired version of the requested master"`
	Prefix         string `json:"prefix,omitempty" description:"hostname prefix for new workers"`
	DiskEncryption bool   `json:"diskEncryption" description:"disable encryption on a worker"`
	EnableTrusted  bool   `json:"enableTrusted" description:"Set to true if trusted hardware should be requested"`
}

// ServiceBindRequest ...
type ServiceBindRequest struct {
	ClusterNameOrID         string
	SpaceGUID               string `json:"spaceGUID" binding:"required"`
	ServiceInstanceNameOrID string `json:"serviceInstanceGUID" binding:"required"`
	NamespaceID             string `json:"namespaceID" binding:"required"`
}

// ServiceBindResponse ...
type ServiceBindResponse struct {
	ServiceInstanceGUID string `json:"serviceInstanceGUID" binding:"required"`
	NamespaceID         string `json:"namespaceID" binding:"required"`
	SecretName          string `json:"secretName"`
	Binding             string `json:"binding"`
}

//BoundService ...
type BoundService struct {
	ServiceName    string `json:"servicename"`
	ServiceID      string `json:"serviceid"`
	ServiceKeyName string `json:"servicekeyname"`
	Namespace      string `json:"namespace"`
}

type BoundServices []BoundService

//Clusters interface
type Clusters interface {
	Create(params ClusterCreateRequest, target ClusterTargetHeader) (ClusterCreateResponse, error)
	List(target ClusterTargetHeader) ([]ClusterInfo, error)
	Update(name string, params ClusterUpdateParam, target ClusterTargetHeader) error
	Delete(name string, target ClusterTargetHeader) error
	Find(name string, target ClusterTargetHeader) (ClusterInfo, error)
	GetClusterConfig(name, homeDir string, admin bool, target ClusterTargetHeader) (string, error)
	UnsetCredentials(target ClusterTargetHeader) error
	SetCredentials(slUsername, slAPIKey string, target ClusterTargetHeader) error
	BindService(params ServiceBindRequest, target ClusterTargetHeader) (ServiceBindResponse, error)
	UnBindService(clusterNameOrID, namespaceID, serviceInstanceGUID string, target ClusterTargetHeader) error
	ListServicesBoundToCluster(clusterNameOrID, namespace string, target ClusterTargetHeader) (BoundServices, error)
	FindServiceBoundToCluster(clusterNameOrID, serviceName, namespace string, target ClusterTargetHeader) (BoundService, error)
}

type clusters struct {
	client *client.Client
}

func newClusterAPI(c *client.Client) Clusters {
	return &clusters{
		client: c,
	}
}

//Create ...
func (r *clusters) Create(params ClusterCreateRequest, target ClusterTargetHeader) (ClusterCreateResponse, error) {
	var cluster ClusterCreateResponse
	_, err := r.client.Post("/v1/clusters", params, &cluster, target.ToMap())
	return cluster, err
}

//Update ...
func (r *clusters) Update(name string, params ClusterUpdateParam, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s", name)
	_, err := r.client.Put(rawURL, params, nil, target.ToMap())
	return err
}

//Delete ...
func (r *clusters) Delete(name string, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s", name)
	_, err := r.client.Delete(rawURL, target.ToMap())
	return err
}

//List ...
func (r *clusters) List(target ClusterTargetHeader) ([]ClusterInfo, error) {
	clusters := []ClusterInfo{}
	_, err := r.client.Get("/v1/clusters", &clusters, target.ToMap())
	if err != nil {
		return nil, err
	}

	return clusters, err
}

//Find ...
func (r *clusters) Find(name string, target ClusterTargetHeader) (ClusterInfo, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s?showResources=true", name)
	cluster := ClusterInfo{}
	_, err := r.client.Get(rawURL, &cluster, target.ToMap())
	if err != nil {
		return cluster, err
	}

	return cluster, err
}

//GetClusterConfig ...
func (r *clusters) GetClusterConfig(name, dir string, admin bool, target ClusterTargetHeader) (string, error) {
	if !helpers.FileExists(dir) {
		return "", fmt.Errorf("Path: %q, to download the config doesn't exist", dir)
	}
	rawURL := fmt.Sprintf("/v1/clusters/%s/config", name)
	if admin {
		rawURL += "/admin"
	}
	resultDir := ComputeClusterConfigDir(dir, name, admin)
	const kubeConfigName = "config.yml"
	err := os.MkdirAll(resultDir, 0755)
	if err != nil {
		return "", fmt.Errorf("Error creating directory to download the cluster config")
	}
	downloadPath := filepath.Join(resultDir, "config.zip")
	trace.Logger.Println("Will download the kubeconfig at", downloadPath)

	var out *os.File
	if out, err = os.Create(downloadPath); err != nil {
		return "", err
	}
	defer out.Close()
	defer helpers.RemoveFile(downloadPath)
	_, err = r.client.Get(rawURL, out, target.ToMap())
	if err != nil {
		return "", err
	}
	trace.Logger.Println("Downloaded the kubeconfig at", downloadPath)
	if err = helpers.Unzip(downloadPath, resultDir); err != nil {
		return "", err
	}
	defer helpers.RemoveFilesWithPattern(resultDir, "[^(.yml)|(.pem)]$")
	var kubedir, kubeyml string
	files, _ := ioutil.ReadDir(resultDir)
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "kube") {
			kubedir = filepath.Join(resultDir, f.Name())
			files, _ := ioutil.ReadDir(kubedir)
			for _, f := range files {
				old := filepath.Join(kubedir, f.Name())
				new := filepath.Join(kubedir, "../", f.Name())
				if strings.HasSuffix(f.Name(), ".yml") {
					new = filepath.Join(kubedir, "../", kubeConfigName)
					kubeyml = new
				}
				err := os.Rename(old, new)
				if err != nil {
					return "", fmt.Errorf("Couldn't rename: %q", err)
				}
			}
			break
		}
	}
	if kubedir == "" {
		return "", errors.New("Unable to locate kube config in zip archive")
	}
	return filepath.Abs(kubeyml)
}

//UnsetCredentials ...
func (r *clusters) UnsetCredentials(target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/credentials")
	_, err := r.client.Delete(rawURL, target.ToMap())
	return err
}

//SetCredentials ...
func (r *clusters) SetCredentials(slUsername, slAPIKey string, target ClusterTargetHeader) error {
	slHeader := &ClusterSoftlayerHeader{
		SoftLayerAPIKey:   slAPIKey,
		SoftLayerUsername: slUsername,
	}
	_, err := r.client.Post("/v1/credentials", nil, nil, target.ToMap(), slHeader.ToMap())
	return err
}

//BindService ...
func (r *clusters) BindService(params ServiceBindRequest, target ClusterTargetHeader) (ServiceBindResponse, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s/services", params.ClusterNameOrID)
	payLoad := struct {
		SpaceGUID               string `json:"spaceGUID" binding:"required"`
		ServiceInstanceNameOrID string `json:"serviceInstanceGUID" binding:"required"`
		NamespaceID             string `json:"namespaceID" binding:"required"`
	}{
		SpaceGUID:               params.SpaceGUID,
		ServiceInstanceNameOrID: params.ServiceInstanceNameOrID,
		NamespaceID:             params.NamespaceID,
	}
	var cluster ServiceBindResponse
	_, err := r.client.Post(rawURL, payLoad, &cluster, target.ToMap())
	return cluster, err
}

//UnBindService ...
func (r *clusters) UnBindService(clusterNameOrID, namespaceID, serviceInstanceGUID string, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/services/%s/%s", clusterNameOrID, namespaceID, serviceInstanceGUID)
	_, err := r.client.Delete(rawURL, target.ToMap())
	return err
}

//ComputeClusterConfigDir ...
func ComputeClusterConfigDir(dir, name string, admin bool) string {
	resultDirPrefix := name
	resultDirSuffix := "_k8sconfig"
	if len(name) < 30 {
		//Make it longer for uniqueness
		h := sha1.New()
		h.Write([]byte(name))
		resultDirPrefix = fmt.Sprintf("%x_%s", h.Sum(nil), name)
	}
	if admin {
		resultDirPrefix = fmt.Sprintf("%s_admin", resultDirPrefix)
	}
	resultDir := filepath.Join(dir, fmt.Sprintf("%s%s", resultDirPrefix, resultDirSuffix))
	return resultDir
}

//ListServicesBoundToCluster ...
func (r *clusters) ListServicesBoundToCluster(clusterNameOrID, namespace string, target ClusterTargetHeader) (BoundServices, error) {
	var boundServices BoundServices
	var path string

	if namespace == "" {
		path = fmt.Sprintf("/v1/clusters/%s/services", clusterNameOrID)

	} else {
		path = fmt.Sprintf("/v1/clusters/%s/services/%s", clusterNameOrID, namespace)
	}
	_, err := r.client.Get(path, &boundServices, target.ToMap())
	if err != nil {
		return boundServices, err
	}

	return boundServices, err
}

//FindServiceBoundToCluster...
func (r *clusters) FindServiceBoundToCluster(clusterNameOrID, serviceNameOrId, namespace string, target ClusterTargetHeader) (BoundService, error) {
	var boundService BoundService
	boundServices, err := r.ListServicesBoundToCluster(clusterNameOrID, namespace, target)
	if err != nil {
		return boundService, err
	}
	for _, boundService := range boundServices {
		if strings.Compare(boundService.ServiceName, serviceNameOrId) == 0 || strings.Compare(boundService.ServiceID, serviceNameOrId) == 0 {
			return boundService, nil
		}
	}

	return boundService, err
}
