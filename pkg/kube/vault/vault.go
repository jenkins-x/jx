package vault

import (
	"fmt"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/cluster"
	"github.com/jenkins-x/jx/pkg/kube/serviceaccount"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	BankVaultsOperatorImage = "banzaicloud/vault-operator"
	BankVaultsImage         = "banzaicloud/bank-vaults"
	BankVaultsImageTag      = "0.4.7"
	defaultNumVaults        = 1
	vaultImage              = "vault"
	vaultImageTag           = "0.11.6"
	gcpServiceAccountEnv    = "GOOGLE_APPLICATION_CREDENTIALS"
	gcpServiceAccountPath   = "/etc/gcp/service-account.json"

	awsServiceAccountEnv  = "AWS_SHARED_CREDENTIALS_FILE"
	awsServiceAccountPath = "/etc/aws/credentials"

	vaultAuthName = "auth"
	vaultAuthType = "kubernetes"
	vaultAuthTTL  = "1h"

	vaultRoleName = "vault-auth"
)

// Vault stores some details of a Vault resource
type Vault struct {
	Name                   string
	Namespace              string
	URL                    string
	AuthServiceAccountName string
}

// GCPConfig keeps the configuration for Google Cloud
type GCPConfig struct {
	ProjectId   string
	KmsKeyring  string
	KmsKey      string
	KmsLocation string
	GcsBucket   string
}

// GCSConfig Google Cloud Storage config for Vault backend
type GCSConfig struct {
	Bucket    string `json:"bucket"`
	HaEnabled string `json:"ha_enabled"`
}

// AWSConfig keeps the vault configuration for AWS
type AWSConfig struct {
	v1alpha1.AWSUnsealConfig
	DynamoDBTable   string
	DynamoDBRegion  string
	AccessKeyID     string
	SecretAccessKey string
}

// DynamoDBConfig AWS DynamoDB config for Vault backend
type DynamoDBConfig struct {
	HaEnabled       string `json:"ha_enabled"`
	Region          string `json:"region"`
	Table           string `json:"table"`
	AccessKeyID     string `json:"access_key"`
	SecretAccessKey string `json:"secret_key"`
}

// VaultAuths list of vault authentications
type VaultAuths []VaultAuth

// VaultAuth vault auth configuration
type VaultAuth struct {
	Roles []VaultRole `json:"roles"`
	Type  string      `json:"type"`
}

// VaultRole role configuration for VaultAuth
type VaultRole struct {
	BoundServiceAccountNames      string `json:"bound_service_account_names"`
	BoundServiceAccountNamespaces string `json:"bound_service_account_namespaces"`
	Name                          string `json:"name"`
	Policies                      string `json:"policies"`
	TTL                           string `json:"ttl"`
}

// VaultPolicies list of vault policies
type VaultPolicies []VaultPolicy

// VaultPolicy vault policy
type VaultPolicy struct {
	Name  string `json:"name"`
	Rules string `json:"rules"`
}

// Tcp address for vault server
type Tcp struct {
	Address    string `json:"address"`
	TlsDisable bool   `json:"tls_disable"`
}

// Listener vault server listener
type Listener struct {
	Tcp Tcp `json:"tcp"`
}

// Telemetry address for telemetry server
type Telemetry struct {
	StatsdAddress string `json:"statsd_address"`
}

// Storage configuration for Vault storage
type Storage struct {
	GCS      *GCSConfig      `json:"gcs,omitempty"`
	DynamoDB *DynamoDBConfig `json:"dynamodb,omitempty"`
}

// SystemVaultName returns the name of the system vault based on the cluster name
func SystemVaultName(kuber kube.Kuber) (string, error) {
	clusterName, err := cluster.ShortName(kuber)
	if err != nil {
		return "", err
	}
	return SystemVaultNameForCluster(clusterName), nil
}

// SystemVaultNameForCluster returns the system vault name from a given cluster name
func SystemVaultNameForCluster(clusterName string) string {
	shortClusterName := cluster.ShortClusterName(clusterName)
	fullName := fmt.Sprintf("%s-%s", vault.SystemVaultNamePrefix, shortClusterName)
	return cluster.ShortNameN(fullName, 22)
}

// CreateGKEVault creates a new vault backed by GCP KMS and storage
func CreateGKEVault(kubeClient kubernetes.Interface, vaultOperatorClient versioned.Interface, name string, ns string,
	gcpServiceAccountSecretName string, gcpConfig *GCPConfig, authServiceAccount string,
	authServiceAccountNamespace string, secretsPathPrefix string) error {

	vault, err := InitializeVault(kubeClient, name, ns, authServiceAccount, authServiceAccountNamespace, secretsPathPrefix)
	if err != nil {
		return err
	}

	vault.Spec.Config["storage"] = Storage{
		GCS: &GCSConfig{
			Bucket:    gcpConfig.GcsBucket,
			HaEnabled: "true",
		},
	}
	vault.Spec.UnsealConfig = v1alpha1.UnsealConfig{
		Google: &v1alpha1.GoogleUnsealConfig{
			KMSKeyRing:    gcpConfig.KmsKeyring,
			KMSCryptoKey:  gcpConfig.KmsKey,
			KMSLocation:   gcpConfig.KmsLocation,
			KMSProject:    gcpConfig.ProjectId,
			StorageBucket: gcpConfig.GcsBucket,
		},
	}
	vault.Spec.CredentialsConfig = v1alpha1.CredentialsConfig{
		Env:        gcpServiceAccountEnv,
		Path:       gcpServiceAccountPath,
		SecretName: gcpServiceAccountSecretName,
	}

	_, err = vaultOperatorClient.VaultV1alpha1().Vaults(ns).Create(vault)
	return err
}

// CreateAWSVault creates a new vault backed by AWS KMS and DynamoDB storage
func CreateAWSVault(kubeClient kubernetes.Interface, vaultOperatorClient versioned.Interface, name string, ns string,
	awsServiceAccountSecretName string, awsConfig *AWSConfig, authServiceAccount string,
	authServiceAccountNamespace string, secretsPathPrefix string) error {

	vault, err := InitializeVault(kubeClient, name, ns, authServiceAccount, authServiceAccountNamespace, secretsPathPrefix)
	if err != nil {
		return err
	}

	vault.Spec.Config["storage"] = Storage{
		DynamoDB: &DynamoDBConfig{
			HaEnabled:       "true",
			Region:          awsConfig.DynamoDBRegion,
			Table:           awsConfig.DynamoDBTable,
			AccessKeyID:     awsConfig.AccessKeyID,
			SecretAccessKey: awsConfig.SecretAccessKey,
		},
	}
	vault.Spec.UnsealConfig = v1alpha1.UnsealConfig{
		AWS: &awsConfig.AWSUnsealConfig,
	}
	vault.Spec.CredentialsConfig = v1alpha1.CredentialsConfig{
		Env:        awsServiceAccountEnv,
		Path:       awsServiceAccountPath,
		SecretName: awsServiceAccountSecretName,
	}

	_, err = vaultOperatorClient.VaultV1alpha1().Vaults(ns).Create(vault)
	return err
}

// InitializeVault intializes and returns vault struct
func InitializeVault(kubeClient kubernetes.Interface, name string, ns string, authServiceAccount string,
	authServiceAccountNamespace string, secretsPathPrefix string) (*v1alpha1.Vault, error) {

	err := createVaultServiceAccount(kubeClient, ns, name)
	if err != nil {
		return nil, errors.Wrapf(err, "creating the vault service account '%s'", name)
	}

	err = ensureVaultRoleBinding(kubeClient, ns, vaultRoleName, name, name)
	if err != nil {
		return nil, errors.Wrapf(err, "ensuring vault cluster role binding '%s' is created", name)
	}

	if secretsPathPrefix == "" {
		secretsPathPrefix = vault.DefaultSecretsPathPrefix
	}
	pathRule := &vault.PathRule{
		Path: []vault.PathPolicy{{
			Prefix:       secretsPathPrefix,
			Capabilities: vault.DefaultSecretsCapabiltities,
		}},
	}
	vaultRule, err := pathRule.String()
	if err != nil {
		return nil, errors.Wrap(err, "encoding the policies for secret path")
	}

	vault := &v1alpha1.Vault{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Vault",
			APIVersion: "vault.banzaicloud.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.VaultSpec{
			Size:            defaultNumVaults,
			Image:           vaultImage + ":" + vaultImageTag,
			BankVaultsImage: BankVaultsImage + ":" + BankVaultsImageTag,
			ServiceType:     string(v1.ServiceTypeClusterIP),
			ServiceAccount:  name,
			Config: map[string]interface{}{
				"api_addr":           fmt.Sprintf("http://%s.%s:8200", name, ns),
				"disable_clustering": true,
				"listener": Listener{
					Tcp: Tcp{
						Address:    "0.0.0.0:8200",
						TlsDisable: true,
					},
				},
				"telemetry": Telemetry{
					StatsdAddress: "localhost:9125",
				},
				"ui": true,
			},
			ExternalConfig: map[string]interface{}{
				vaultAuthName: []VaultAuth{
					{
						Roles: []VaultRole{
							{

								BoundServiceAccountNames:      authServiceAccount,
								BoundServiceAccountNamespaces: authServiceAccountNamespace,
								Name:                          authServiceAccount,
								Policies:                      vault.PathRulesName,
								TTL:                           vaultAuthTTL,
							},
						},
						Type: vaultAuthType,
					},
				},
				vault.PoliciesName: []VaultPolicy{
					{
						Name:  vault.PathRulesName,
						Rules: vaultRule,
					},
				},
			},
		},
	}

	return vault, err
}

func createVaultServiceAccount(client kubernetes.Interface, namespace string, name string) error {
	_, err := serviceaccount.CreateServiceAccount(client, namespace, name)
	if err != nil {
		return errors.Wrap(err, "creating vault service account")
	}
	return nil
}

func ensureVaultRoleBinding(client kubernetes.Interface, namespace string, roleName string,
	roleBindingName string, serviceAccount string) error {
	apiGroups := []string{"authentication.k8s.io"}
	resources := []string{"tokenreviews"}
	verbs := []string{"*"}
	found := kube.IsClusterRole(client, roleName)
	if found {
		err := kube.DeleteClusterRole(client, roleName)
		if err != nil {
			return errors.Wrapf(err, "deleting the existing cluster role '%s'", roleName)
		}
	}
	err := kube.CreateClusterRole(client, namespace, roleName, apiGroups, resources, verbs)
	if err != nil {
		return errors.Wrapf(err, "creating the cluster role '%s' for vault", roleName)
	}

	found = kube.IsClusterRoleBinding(client, roleBindingName)
	if found {
		err := kube.DeleteClusterRoleBinding(client, roleBindingName)
		if err != nil {
			return errors.Wrapf(err, "deleting the existing cluster role binding '%s'", roleBindingName)
		}
	}

	err = kube.CreateClusterRoleBinding(client, namespace, roleBindingName, serviceAccount, roleName)
	if err != nil {
		return errors.Wrapf(err, "creating the cluster role binding '%s' for vault", roleBindingName)
	}
	return nil
}

// FindVault checks if a vault is available
func FindVault(vaultOperatorClient versioned.Interface, name string, ns string) bool {
	_, err := GetVault(vaultOperatorClient, name, ns)
	if err != nil {
		return false
	}
	return true
}

// GetVault gets a specific vault
func GetVault(vaultOperatorClient versioned.Interface, name string, ns string) (*v1alpha1.Vault, error) {
	return vaultOperatorClient.Vault().Vaults(ns).Get(name, metav1.GetOptions{})
}

// GetVaults returns all vaults available in a given namespaces
func GetVaults(client kubernetes.Interface, vaultOperatorClient versioned.Interface, ns string) ([]*Vault, error) {
	vaultList, err := vaultOperatorClient.Vault().Vaults(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "listing vaults in namespace '%s'", ns)
	}

	vaults := []*Vault{}
	for _, v := range vaultList.Items {
		vaultName := v.Name
		vaultAuthSaName := GetAuthSaName(v)
		vaultURL, err := services.FindServiceURL(client, ns, vaultName)
		if err != nil {
			vaultURL = ""
		}
		vault := Vault{
			Name:                   vaultName,
			Namespace:              ns,
			URL:                    vaultURL,
			AuthServiceAccountName: vaultAuthSaName,
		}
		vaults = append(vaults, &vault)
	}
	return vaults, nil
}

// DeleteVault delete a Vault resource
func DeleteVault(vaultOperatorClient versioned.Interface, name string, ns string) error {
	return vaultOperatorClient.Vault().Vaults(ns).Delete(name, &metav1.DeleteOptions{})
}

// GetAuthSaName gets the Auth Service Account name for the vault
func GetAuthSaName(vault v1alpha1.Vault) string {
	// This is nasty, but the ExternalConfig member of VaultSpec is just defined as a map[string]interface{} :-(
	authArray := vault.Spec.ExternalConfig["auth"]
	authObject := authArray.([]interface{})[0]
	roleArray := authObject.(map[string]interface{})["roles"]
	roleObject := roleArray.([]interface{})[0]
	name := roleObject.(map[string]interface{})["name"]

	return name.(string)
}
