package kube

import (
	"fmt"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNumVaults      = 2
	vaultImage            = "vault:0.11.2"
	bankVaultsImage       = "banzaicloud/bank-vaults:latest"
	gcpServiceAccountEnv  = "GOOGLE_APPLICATION_CREDENTIALS"
	gcpServiceAccountPath = "/etc/gcp/service-account.json"

	vaultPoliciesName    = "policies"
	vaultRuleSecretsName = "allow_secrets"
	vaultRuleSecrets     = "path \"secret/*\" { capabilities = [\"create\", \"read\", \"update\", \"delete\", \"list\"] }"
	vaultAuthName        = "auth"
	vaultAuthType        = "kubernetes"
	vaultAuthTTL         = "1h"
)

// GCPConfig keeps the configuration for Google Cloud
type GCPConfig struct {
	ProjectId   string
	KmsKeyring  string
	KmsKey      string
	KmsLocation string
	GcsBucket   string
}

type GCSConfig struct {
	Bucket    string `json:"bucket"`
	HaEnabled string `json:"ha_enabled"`
}

type VaultAuths []VaultAuth

type VaultAuth struct {
	Roles []VaultRole `json:"roles"`
	Type  string      `json:"type"`
}

type VaultRole struct {
	BoundServiceAccountNames      string `json:"bound_service_account_names"`
	BoundServiceAccountNamespaces string `json:"bound_service_account_namespaces"`
	Name                          string `json:"name"`
	Policies                      string `json:"policies"`
	TTL                           string `json:"ttl"`
}

type VaultPolicies []VaultPolicy

type VaultPolicy struct {
	Name  string
	Rules string
}

type Tcp struct {
	Address    string `json:"address"`
	TlsDisable bool   `json:"tls_disable"`
}

type Listener struct {
	Tcp Tcp `json:"tcp"`
}

type Telemetry struct {
	StatsdAddress string `json:"statsd_address"`
}

type Storage struct {
	GCS GCSConfig `json:"gcs"`
}

// CreateVault creates a new vault
func CreateVault(vaultOperatorClient versioned.Interface, name string, ns string,
	gcpServiceAccountSecretName string, gcpConfig *GCPConfig, authServiceAccount string,
	authServiceAccountNamespace string) error {
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
			Image:           vaultImage,
			BankVaultsImage: bankVaultsImage,
			Config: map[string]interface{}{
				"api_addr":           fmt.Sprintf("http://%s.%s:8200", name, ns),
				"disable_clustering": true,
				"listener": Listener{
					Tcp: Tcp{
						Address:    "0.0.0.0:8200",
						TlsDisable: true,
					},
				},
				"storage": Storage{
					GCS: GCSConfig{
						Bucket:    gcpConfig.GcsBucket,
						HaEnabled: "true",
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
								Policies:                      vaultRuleSecretsName,
								TTL:                           vaultAuthTTL,
							},
						},
						Type: vaultAuthType,
					},
				},
				vaultPoliciesName: []VaultPolicy{
					{
						Name:  vaultRuleSecretsName,
						Rules: vaultRuleSecrets,
					},
				},
			},
			UnsealConfig: v1alpha1.UnsealConfig{
				Google: &v1alpha1.GoogleUnsealConfig{
					KMSKeyRing:    gcpConfig.KmsKeyring,
					KMSCryptoKey:  gcpConfig.KmsKey,
					KMSLocation:   gcpConfig.KmsLocation,
					KMSProject:    gcpConfig.ProjectId,
					StorageBucket: gcpConfig.GcsBucket,
				},
			},
			CredentialsConfig: v1alpha1.CredentialsConfig{
				Env:        gcpServiceAccountEnv,
				Path:       gcpServiceAccountPath,
				SecretName: gcpServiceAccountSecretName,
			},
		},
	}

	_, err := vaultOperatorClient.Vault().Vaults(ns).Create(vault)
	return err
}
