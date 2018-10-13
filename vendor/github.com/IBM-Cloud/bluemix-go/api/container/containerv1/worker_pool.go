package containerv1

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/client"
)

// WorkerPool common worker pool data
type WorkerPoolConfig struct {
	Name        string            `json:"name" binding:"required"`
	Size        int               `json:"sizePerZone" binding:"required"`
	MachineType string            `json:"machineType" binding:"required"`
	Isolation   string            `json:"isolation"`
	Labels      map[string]string `json:"labels"`
}

// WorkerPoolRequest provides worker pool data
// swagger:model
type WorkerPoolRequest struct {
	WorkerPoolConfig
	DiskEncryption bool             `json:"diskEncryption" description:"true or false to use encryption for the secondary disk"`
	Zones          []WorkerPoolZone `json:"zones"`
}

// WorkerPoolPatchRequest provides attributes to patch update worker pool
// swagger:model
type WorkerPoolPatchRequest struct {
	Size            int    `json:"sizePerZone"`
	ReasonForResize string `json:"reasonForResize"`
	State           string `json:"state"`
}

// WorkerPoolResponse provides worker pool data
// swagger:model
type WorkerPoolResponse struct {
	WorkerPoolConfig
	ID              string                  `json:"id" binding:"required"`
	Region          string                  `json:"region" binding:"required"`
	State           string                  `json:"state"`
	ReasonForDelete string                  `json:"reasonForDelete"`
	IsBalanced      bool                    `json:"isBalanced"`
	Zones           WorkerPoolZoneResponses `json:"zones"`
}

// WorkerPoolResponses sorts WorkerPoolResponse by ID.
// swagger:model
type WorkerPoolResponses []WorkerPoolResponse

// WorkerPoolZoneNetwork holds network configuration for a zone
type WorkerPoolZoneNetwork struct {
	PrivateVLAN string `json:"privateVlan" binding:"required"`
	PublicVLAN  string `json:"publicVlan"`
}

// WorkerPoolZone provides zone data
// swagger:model
type WorkerPoolZone struct {
	WorkerPoolZoneNetwork
	ID string `json:"id" binding:"required"`
}

// WorkerPoolZonePatchRequest updates worker pool zone data
// swagger:model
type WorkerPoolZonePatchRequest struct {
	WorkerPoolZoneNetwork
}

// WorkerPoolZoneResponse response contents for zone
// swagger:model
type WorkerPoolZoneResponse struct {
	WorkerPoolZone
	WorkerCount int `json:"workerCount"`
}

// WorkerPoolZoneResponses sorts WorkerPoolZoneResponse by ID.
// swagger:model
type WorkerPoolZoneResponses []WorkerPoolZoneResponse

//Workers ...
type WorkerPool interface {
	CreateWorkerPool(clusterNameOrID string, workerPoolReq WorkerPoolRequest, target ClusterTargetHeader) (WorkerPoolResponse, error)
	ResizeWorkerPool(clusterNameOrID, workerPoolNameOrID string, size int, target ClusterTargetHeader) error
	PatchWorkerPool(clusterNameOrID, workerPoolNameOrID, state string, target ClusterTargetHeader) error
	DeleteWorkerPool(clusterNameOrID string, workerPoolNameOrID string, target ClusterTargetHeader) error
	ListWorkerPools(clusterNameOrID string, target ClusterTargetHeader) ([]WorkerPoolResponse, error)
	GetWorkerPool(clusterNameOrID, workerPoolNameOrID string, target ClusterTargetHeader) (WorkerPoolResponse, error)
	AddZone(clusterNameOrID string, poolID string, workerPoolZone WorkerPoolZone, target ClusterTargetHeader) error
	RemoveZone(clusterNameOrID, zone, poolID string, target ClusterTargetHeader) error
	UpdateZoneNetwork(clusterNameOrID, zone, poolID, privateVlan, publicVlan string, target ClusterTargetHeader) error
}

type workerpool struct {
	client *client.Client
}

func newWorkerPoolAPI(c *client.Client) WorkerPool {
	return &workerpool{
		client: c,
	}
}

// CreateWorkerPool calls the API to create a worker pool
func (w *workerpool) CreateWorkerPool(clusterNameOrID string, workerPoolReq WorkerPoolRequest, target ClusterTargetHeader) (WorkerPoolResponse, error) {
	var successV WorkerPoolResponse
	_, err := w.client.Post(fmt.Sprintf("/v1/clusters/%s/workerpools", clusterNameOrID), workerPoolReq, &successV, target.ToMap())
	return successV, err
}

// ResizeWorkerPool calls the API to resize a worker
func (w *workerpool) PatchWorkerPool(clusterNameOrID, workerPoolNameOrID, state string, target ClusterTargetHeader) error {
	requestBody := WorkerPoolPatchRequest{
		State: state,
	}
	_, err := w.client.Patch(fmt.Sprintf("/v1/clusters/%s/workerpools/%s", clusterNameOrID, workerPoolNameOrID), requestBody, nil, target.ToMap())
	return err
}

// ResizeWorkerPool calls the API to resize a worker
func (w *workerpool) ResizeWorkerPool(clusterNameOrID, workerPoolNameOrID string, size int, target ClusterTargetHeader) error {
	requestBody := WorkerPoolPatchRequest{
		State: "resizing",
		Size:  size,
	}
	_, err := w.client.Patch(fmt.Sprintf("/v1/clusters/%s/workerpools/%s", clusterNameOrID, workerPoolNameOrID), requestBody, nil, target.ToMap())
	return err
}

// DeleteWorkerPool calls the API to remove a worker pool
func (w *workerpool) DeleteWorkerPool(clusterNameOrID string, workerPoolNameOrID string, target ClusterTargetHeader) error {
	// Make the request, don't care about return value
	_, err := w.client.Delete(fmt.Sprintf("/v1/clusters/%s/workerpools/%s", clusterNameOrID, workerPoolNameOrID), target.ToMap())
	return err
}

// ListWorkerPools calls the API to list all worker pools for a cluster
func (w *workerpool) ListWorkerPools(clusterNameOrID string, target ClusterTargetHeader) ([]WorkerPoolResponse, error) {
	var successV []WorkerPoolResponse
	_, err := w.client.Get(fmt.Sprintf("/v1/clusters/%s/workerpools", clusterNameOrID), &successV, target.ToMap())
	return successV, err
}

// GetWorkerPool calls the API to get a worker pool
func (w *workerpool) GetWorkerPool(clusterNameOrID, workerPoolNameOrID string, target ClusterTargetHeader) (WorkerPoolResponse, error) {
	var successV WorkerPoolResponse
	_, err := w.client.Get(fmt.Sprintf("/v1/clusters/%s/workerpools/%s", clusterNameOrID, workerPoolNameOrID), &successV, target.ToMap())
	return successV, err
}

// AddZone calls the API to add a zone to a cluster and worker pool
func (w *workerpool) AddZone(clusterNameOrID string, poolID string, workerPoolZone WorkerPoolZone, target ClusterTargetHeader) error {
	// Make the request, don't care about return value
	_, err := w.client.Post(fmt.Sprintf("/v1/clusters/%s/workerpools/%s/zones", clusterNameOrID, poolID), workerPoolZone, nil, target.ToMap())
	return err
}

// RemoveZone calls the API to remove a zone from a worker pool in a cluster
func (w *workerpool) RemoveZone(clusterNameOrID, zone, poolID string, target ClusterTargetHeader) error {
	_, err := w.client.Delete(fmt.Sprintf("/v1/clusters/%s/workerpools/%s/zones/%s", clusterNameOrID, poolID, zone), target.ToMap())
	return err
}

// UpdateZoneNetwork calls the API to update a zone's network
func (w *workerpool) UpdateZoneNetwork(clusterNameOrID, zone, poolID, privateVlan, publicVlan string, target ClusterTargetHeader) error {
	body := WorkerPoolZoneNetwork{
		PrivateVLAN: privateVlan,
		PublicVLAN:  publicVlan,
	}
	_, err := w.client.Patch(fmt.Sprintf("/v1/clusters/%s/workerpools/%s/zones/%s", clusterNameOrID, poolID, zone), body, nil, target.ToMap())
	return err
}
