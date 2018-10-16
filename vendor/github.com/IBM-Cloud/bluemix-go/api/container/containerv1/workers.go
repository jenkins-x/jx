package containerv1

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/client"
)

//Worker ...
type Worker struct {
	Billing          string `json:"billing,omitempty"`
	ErrorMessage     string `json:"errorMessage"`
	ID               string `json:"id"`
	Isolation        string `json:"isolation"`
	KubeVersion      string `json:"kubeVersion"`
	MachineType      string `json:"machineType"`
	PrivateIP        string `json:"privateIP"`
	PrivateVlan      string `json:"privateVlan"`
	PublicIP         string `json:"publicIP"`
	PublicVlan       string `json:"publicVlan"`
	Location         string `json:"location"`
	PoolID           string `json:"poolid"`
	PoolName         string `json:"poolName"`
	TrustedStatus    string `json:"trustedStatus"`
	ReasonForDelete  string `json:"reasonForDelete"`
	VersionEOS       string `json:"versionEOS"`
	MasterVersionEOS string `json:"masterVersionEOS"`
	State            string `json:"state"`
	Status           string `json:"status"`
	TargetVersion    string `json:"targetVersion"`
}

//WorkerParam ...
type WorkerParam struct {
	MachineType string `json:"machineType,omitempty" description:"The worker's machine type"`
	PrivateVlan string `json:"privateVlan,omitempty" description:"The worker's private vlan"`
	PublicVlan  string `json:"publicVlan,omitempty" description:"The worker's public vlan"`
	Isolation   string `json:"isolation,omitempty" description:"Can be 'public' or 'private'"`
	WorkerNum   int    `json:"workerNum,omitempty" binding:"required" description:"The number of workers"`
	Prefix      string `json:"prefix,omitempty" description:"hostname prefix for new workers"`
	Action      string `json:"action,omitempty"`
	Count       int    `json:"count,omitempty"`
}

//WorkerUpdateParam ...
type WorkerUpdateParam struct {
	Action string `json:"action" binding:"required" description:"Action to perform of the worker"`
}

//Workers ...
type Workers interface {
	List(clusterName string, target ClusterTargetHeader) ([]Worker, error)
	ListByWorkerPool(clusterIDOrName, workerPoolIDOrName string, showDeleted bool, target ClusterTargetHeader) ([]Worker, error)
	Get(clusterName string, target ClusterTargetHeader) (Worker, error)
	Add(clusterName string, params WorkerParam, target ClusterTargetHeader) error
	Delete(clusterName string, workerD string, target ClusterTargetHeader) error
	Update(clusterName string, workerID string, params WorkerUpdateParam, target ClusterTargetHeader) error
}

type worker struct {
	client *client.Client
}

func newWorkerAPI(c *client.Client) Workers {
	return &worker{
		client: c,
	}
}

//Get ...
func (r *worker) Get(id string, target ClusterTargetHeader) (Worker, error) {
	rawURL := fmt.Sprintf("/v1/workers/%s", id)
	worker := Worker{}
	_, err := r.client.Get(rawURL, &worker, target.ToMap())
	if err != nil {
		return worker, err
	}

	return worker, err
}

func (r *worker) Add(name string, params WorkerParam, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/workers", name)
	_, err := r.client.Post(rawURL, params, nil, target.ToMap())
	return err
}

//Delete ...
func (r *worker) Delete(name string, workerID string, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/workers/%s", name, workerID)
	_, err := r.client.Delete(rawURL, target.ToMap())
	return err
}

//Update ...
func (r *worker) Update(name string, workerID string, params WorkerUpdateParam, target ClusterTargetHeader) error {
	rawURL := fmt.Sprintf("/v1/clusters/%s/workers/%s", name, workerID)
	_, err := r.client.Put(rawURL, params, nil, target.ToMap())
	return err
}

//List ...
func (r *worker) List(name string, target ClusterTargetHeader) ([]Worker, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s/workers", name)
	workers := []Worker{}
	_, err := r.client.Get(rawURL, &workers, target.ToMap())
	if err != nil {
		return nil, err
	}
	return workers, err
}

//ListByWorkerPool ...
func (r *worker) ListByWorkerPool(clusterIDOrName, workerPoolIDOrName string, showDeleted bool, target ClusterTargetHeader) ([]Worker, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s/workers?showDeleted=%t", clusterIDOrName, showDeleted)
	if len(workerPoolIDOrName) > 0 {
		rawURL += "&pool=" + workerPoolIDOrName
	}
	workers := []Worker{}
	_, err := r.client.Get(rawURL, &workers, target.ToMap())
	if err != nil {
		return nil, err
	}
	return workers, err
}
