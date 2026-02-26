package vault

import (
	"fmt"

	"github.com/jenkins-x/jx/v2/pkg/errorutil"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

const (
	// SystemVaultName stores the name of the Vault instance created and managed by Jenkins X unless an external Vault
	// instance is used in which case this name will be empty.
	SystemVaultName = "systemVaultName"

	// URL stores the URL of the external Vault instance if no system internal Vault instance is used.
	URL = "vaultURL"

	// ServiceAccount stores the name of the service account used to connect to Vault.
	ServiceAccount = "vaultServiceAccount"

	// Namespace stores the service account namespace which is allowed to connect to Vault.
	Namespace = "vaultNamespace"

	// SecretEngineMountPoint defines the Vault mount point for the KV secret engine.
	SecretEngineMountPoint = "vaultSecretEngineMountPoint"

	// KubernetesAuthPath defines the path under which the Kubernetes auth method is configured.
	KubernetesAuthPath = "vaultKubernetesAuthPath"

	// DefaultKVEngineMountPoint default mount point for the KV V2 engine
	DefaultKVEngineMountPoint = "secret"

	// DefaultKubernetesAuthPath is the default Kubernetes auth path
	DefaultKubernetesAuthPath = "kubernetes"

	// defaultServiceAccountSuffix is the default suffix appended to the Vault name if no service account name is specified.
	defaultServiceAccountSuffix = "-vt"
)

// Vault stores the required information to connect and authenticate against a Vault instance.
type Vault struct {
	// Name defines the name of the Vault instance, provided we are dealing with an Jenkins X managed Vault instance
	Name string

	// ServiceAccountName is the name of the service account allowed to authenticate against Vault.
	ServiceAccountName string

	// Namespace of the service account authorized to authenticate against Vault.
	Namespace string

	// URL specifies the Vault URL to connect to.
	URL string

	// SecretEngineMountPoint is the mount point to be used for writing data into the KV engine.
	SecretEngineMountPoint string

	// KubernetesAuthPath is the path under which the Vault Kubernetes auth method is configured.
	KubernetesAuthPath string
}

// NewExternalVault creates an external Vault instance configuration from the provided parameters.
func NewExternalVault(url string, serviceAccountName string, namespace string, secretEngineMountPoint string, kubernetesAuthPath string) (Vault, error) {
	if url == "" {
		return Vault{}, errors.New("URL cannot be empty for an external Vault configuration")
	}

	data := map[string]string{}

	data[SystemVaultName] = ""
	data[URL] = url
	data[ServiceAccount] = serviceAccountName
	data[Namespace] = namespace
	data[SecretEngineMountPoint] = secretEngineMountPoint
	data[KubernetesAuthPath] = kubernetesAuthPath

	return FromMap(data, namespace)
}

// NewInternalVault creates an internal Vault instance configuration from the provided parameters.
func NewInternalVault(name string, serviceAccountName string, namespace string) (Vault, error) {
	if name == "" {
		return Vault{}, errors.New("name cannot be empty for an internal Vault configuration")
	}

	if serviceAccountName == "" {
		serviceAccountName = fmt.Sprintf("%s-%s", name, defaultServiceAccountSuffix)
	}

	data := map[string]string{}
	data[SystemVaultName] = name
	data[ServiceAccount] = serviceAccountName
	data[Namespace] = namespace

	return FromMap(data, namespace)
}

// FromMap reads the configuration of a Vault instance from a map.
// defaultNamespace is used when there is no namespace value provided in the map (for backwards compatibility reasons).
func FromMap(data map[string]string, defaultNamespace string) (Vault, error) {
	if data[SystemVaultName] != "" && data[URL] != "" {
		return Vault{}, errors.New("systemVaultName and URL cannot be specified together")
	}

	secretEngineMountPoint := data[SecretEngineMountPoint]
	if secretEngineMountPoint == "" {
		secretEngineMountPoint = DefaultKVEngineMountPoint
	}

	kubernetesAuthPath := data[KubernetesAuthPath]
	if kubernetesAuthPath == "" {
		kubernetesAuthPath = DefaultKubernetesAuthPath
	}

	namespace := data[Namespace]
	if namespace == "" {
		namespace = defaultNamespace
	}

	vault := Vault{
		Name:                   data[SystemVaultName],
		URL:                    data[URL],
		ServiceAccountName:     data[ServiceAccount],
		Namespace:              namespace,
		SecretEngineMountPoint: secretEngineMountPoint,
		KubernetesAuthPath:     kubernetesAuthPath,
	}

	var err error
	external := vault.IsExternal()
	if external {
		err = vault.validateExternalConfiguration()
	} else {
		err = vault.validateInternalConfiguration()
	}

	return vault, err
}

// IsExternal returns true if the Vault instance represents an externally managed Vault instance or one managed by Jenkins X.
func (v *Vault) IsExternal() bool {
	if v.URL == "" {
		return false
	}
	return true
}

// ToMap writes the configuration of this Vault instance to a map
func (v *Vault) ToMap() map[string]string {
	data := map[string]string{}

	data[SystemVaultName] = v.Name
	data[URL] = v.URL
	data[ServiceAccount] = v.ServiceAccountName
	data[Namespace] = v.Namespace
	data[SecretEngineMountPoint] = v.SecretEngineMountPoint
	data[KubernetesAuthPath] = v.KubernetesAuthPath

	return data
}

// validateExternalConfiguration validates the values of the Vault configuration for an external Vault setup.
func (v *Vault) validateExternalConfiguration() error {
	var validationErrors []error
	var err error
	if v.URL == "" {
		return errors.New("URL cannot be empty")
	}

	if v.URL != "" && !util.IsValidUrl(v.URL) {
		err = errors.Errorf("'%s' not a valid URL", v.URL)
		validationErrors = append(validationErrors, err)
	}

	if v.ServiceAccountName == "" {
		err = errors.New("external vault service account name cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	if v.Namespace == "" {
		err = errors.New("external vault namespace cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	if v.SecretEngineMountPoint == "" {
		err = errors.New("external vault secret engine mount point cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	if v.KubernetesAuthPath == "" {
		err = errors.New("external vault Kubernetes auth path cannot be empty")
		validationErrors = append(validationErrors, err)
	}
	return errorutil.CombineErrors(validationErrors...)
}

// validateInternalConfiguration validates the values of the Vault configuration  for an internal, Jenkins X managed, Vault setup.
func (v *Vault) validateInternalConfiguration() error {
	var validationErrors []error
	var err error
	if v.Name == "" {
		err = errors.New("internal vault name cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	if v.ServiceAccountName == "" {
		err = errors.New("internal vault service account name cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	if v.Namespace == "" {
		err = errors.New("internal vault namespace cannot be empty")
		validationErrors = append(validationErrors, err)
	}

	return errorutil.CombineErrors(validationErrors...)
}
