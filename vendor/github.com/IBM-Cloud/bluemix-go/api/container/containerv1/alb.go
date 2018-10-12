package containerv1

import (
	"fmt"
	"github.com/IBM-Cloud/bluemix-go/client"
)

type ClusterALB struct {
	ID                string      `json:"id"`
	Region            string      `json:"region"`
	DataCenter        string      `json:"dataCenter"`
	IsPaid            bool        `json:"isPaid"`
	IngressHostname   string      `json:"ingressHostname"`
	IngressSecretName string      `json:"ingressSecretName"`
	ALBs              []ALBConfig `json:"alb"`
}

// ALBConfig config for alb configuration
// swagger:model
type ALBConfig struct {
	ALBID             string `json:"albID" description:"The ALB id"`
	ClusterID         string `json:"clusterID"`
	Name              string `json:"name"`
	ALBType           string `json:"albType"`
	Enable            bool   `json:"enable" description:"Enable (true) or disable(false) ALB"`
	State             string `json:"state"`
	CreatedDate       string `json:"createdDate"`
	NumOfInstances    string `json:"numOfInstances" description:"Desired number of ALB replicas"`
	Resize            bool   `json:"resize" description:"Indicate whether resizing should be done"`
	ALBIP             string `json:"albip" description:"BYOIP VIP to use for ALB. Currently supported only for private ALB"`
	Zone              string `json:"zone" description:"Zone to use for adding ALB. This is indicative of the AZ in which ALB will be deployed"`
	DisableDeployment bool   `json:"disableDeployment" description:"Indicate whether to disable deployment only on disable alb"`
}

// ClusterALBSecret albsecret related information for cluster
// swagger:model
type ClusterALBSecret struct {
	ID         string            `json:"id"`
	Region     string            `json:"region"`
	DataCenter string            `json:"dataCenter"`
	IsPaid     bool              `json:"isPaid"`
	ALBSecrets []ALBSecretConfig `json:"albSecrets" description:"All the ALB secrets created in this cluster"`
}

// ALBSecretConfig config for alb-secret configuration
// swagger:model
type ALBSecretConfig struct {
	SecretName          string `json:"secretName" description:"Name of the ALB secret"`
	ClusterID           string `json:"clusterID"`
	DomainName          string `json:"domainName" description:"Domain name of the certficate"`
	CloudCertInstanceID string `json:"cloudCertInstanceID" description:"Cloud Cert instance ID from which certficate is downloaded"`
	ClusterCrn          string `json:"clusterCrn"`
	CertCrn             string `json:"certCrn" description:"Unique CRN of the certficate which can be located in cloud cert instance"`
	IssuerName          string `json:"issuerName" description:"Issuer name of the certficate"`
	ExpiresOn           string `json:"expiresOn" description:"Expiry date of the certficate"`
	State               string `json:"state" description:"State of ALB secret"`
}

// ALBSecretsPerCRN ...
type ALBSecretsPerCRN struct {
	ALBSecrets []string `json:"albsecrets" description:"ALB secrets correponding to a CRN"`
}

//Clusters interface
type Albs interface {
	ListClusterALBs(clusterNameOrID string, target ClusterTargetHeader) ([]ALBConfig, error)
	GetALB(albID string, target ClusterTargetHeader) (ALBConfig, error)
	ConfigureALB(albID string, config ALBConfig, target ClusterTargetHeader) error
	RemoveALB(albID string, target ClusterTargetHeader) error
	DeployALBCert(config ALBSecretConfig, target ClusterTargetHeader) error
	UpdateALBCert(config ALBSecretConfig, target ClusterTargetHeader) error
	RemoveALBCertBySecretName(clusterID string, secretName string, target ClusterTargetHeader) error
	RemoveALBCertByCertCRN(clusterID string, certCRN string, target ClusterTargetHeader) error
	GetClusterALBCertBySecretName(clusterID string, secretName string, target ClusterTargetHeader) (ALBSecretConfig, error)
	GetClusterALBCertByCertCRN(clusterID string, certCRN string, target ClusterTargetHeader) (ALBSecretConfig, error)
	ListALBCerts(clusterID string, target ClusterTargetHeader) ([]ALBSecretConfig, error)
	GetALBTypes(target ClusterTargetHeader) ([]string, error)
}

type alb struct {
	client *client.Client
}

func newAlbAPI(c *client.Client) Albs {
	return &alb{
		client: c,
	}
}

// ListClusterALBs returns the list of albs available for cluster
func (r *alb) ListClusterALBs(clusterNameOrID string, target ClusterTargetHeader) ([]ALBConfig, error) {
	var successV ClusterALB
	rawURL := fmt.Sprintf("/v1/alb/clusters/%s", clusterNameOrID)
	_, err := r.client.Get(rawURL, &successV, target.ToMap())
	return successV.ALBs, err
}

// GetALB returns details about particular alb
func (r *alb) GetALB(albID string, target ClusterTargetHeader) (ALBConfig, error) {
	var successV ALBConfig
	_, err := r.client.Get(fmt.Sprintf("/v1/alb/albs/%s", albID), &successV, target.ToMap())
	return successV, err
}

// ConfigureALB enables or disables alb for cluster
func (r *alb) ConfigureALB(albID string, config ALBConfig, target ClusterTargetHeader) error {
	var successV interface{}
	if config.Enable {
		_, err := r.client.Post("/v1/alb/albs", config, &successV, target.ToMap())
		return err
	}
	_, err := r.client.Delete(fmt.Sprintf("/v1/alb/albs/%s", albID), target.ToMap())
	return err
}

// RemoveALB removes the alb
func (r *alb) RemoveALB(albID string, target ClusterTargetHeader) error {
	_, err := r.client.Delete(fmt.Sprintf("/v1/alb/albs/%s", albID), target.ToMap())
	return err
}

// DeployALBCert deploys alb-cert
func (r *alb) DeployALBCert(config ALBSecretConfig, target ClusterTargetHeader) error {
	var successV interface{}
	_, err := r.client.Post("/v1/alb/albsecrets", config, &successV, target.ToMap())
	return err
}

// UpdateALBCert updates alb-cert
func (r *alb) UpdateALBCert(config ALBSecretConfig, target ClusterTargetHeader) error {
	_, err := r.client.Put("/v1/alb/albsecrets", config, nil, target.ToMap())
	return err
}

// RemoveALBCertBySecretName removes the alb-cert
func (r *alb) RemoveALBCertBySecretName(clusterID string, secretName string, target ClusterTargetHeader) error {
	_, err := r.client.Delete(fmt.Sprintf("/v1/alb/clusters/%s/albsecrets?albSecretName=%s", clusterID, secretName), target.ToMap())
	return err
}

// RemoveALBCertByCertCRN removes the alb-cert
func (r *alb) RemoveALBCertByCertCRN(clusterID string, certCRN string, target ClusterTargetHeader) error {
	_, err := r.client.Delete(fmt.Sprintf("/v1/alb/clusters/%s/albsecrets?certCrn=%s", clusterID, certCRN), target.ToMap())
	return err
}

// GetClusterALBCertBySecretName returns details about specified alb cert for given secretName
func (r *alb) GetClusterALBCertBySecretName(clusterID string, secretName string, target ClusterTargetHeader) (ALBSecretConfig, error) {
	var successV ALBSecretConfig
	_, err := r.client.Get(fmt.Sprintf("/v1/alb/clusters/%s/albsecrets?albSecretName=%s", clusterID, secretName), &successV, target.ToMap())
	return successV, err
}

// GetClusterALBCertByCertCrn returns details about specified alb cert for given certCRN
func (r *alb) GetClusterALBCertByCertCRN(clusterID string, certCRN string, target ClusterTargetHeader) (ALBSecretConfig, error) {
	var successV ALBSecretConfig
	_, err := r.client.Get(fmt.Sprintf("/v1/alb/clusters/%s/albsecrets?certCrn=%s", clusterID, certCRN), &successV, target.ToMap())
	return successV, err
}

// ListALBCerts for cluster...
func (r *alb) ListALBCerts(clusterID string, target ClusterTargetHeader) ([]ALBSecretConfig, error) {
	var successV ClusterALBSecret
	_, err := r.client.Get(fmt.Sprintf("/v1/alb/clusters/%s/albsecrets", clusterID), &successV, target.ToMap())
	return successV.ALBSecrets, err
}

// GetALBTypes returns list of available alb types
func (r *alb) GetALBTypes(target ClusterTargetHeader) ([]string, error) {
	var successV []string
	_, err := r.client.Get("/v1/alb/albtypes", &successV, target.ToMap())
	return successV, err
}
