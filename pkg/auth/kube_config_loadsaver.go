package auth

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// labelKind inidicates the kind of auth such as git
	labelKind = "jenkins.io/kind"
	// labelServiceKind inidicates the  kind of service such as github
	labelServiceKind = "jenkins.io/service-kind"
	// labelCreatedBy indicates the service that created this resource
	labelCreatedBy = "jenkins.io/created-by"
	// labelCredentialsType the kind of jenkins credential for a secret
	labelCredentialsType = "jenkins.io/credentials-type"
	// valueCreatedByJX for resources created by the Jenkins X CLI
	valueCreatedByJX = "jx"
	// valueCredentialTypeUsernamePassword for user password credential secrets
	valueCredentialTypeUsernamePassword = "usernamePassword"
	// annotationURL indicates a service/server's URL
	annotationURL = "jenkins.io/url"
	// annotationName indicates a service/server's textual name (can be mixed case, contain spaces unlike Kubernetes resources)
	annotationName = "jenkins.io/name"
	// annotationCredentialsDescription the description text for a Credential on a Secret
	annotationCredentialsDescription = "jenkins.io/credentials-description"
	// secretDataUsername the username in a Secret/Credentials
	usernameKey = "username"
	// secretDataPassword the password in a Secret/Credentials
	passwordKey = "password"
	// secretPrefix prefix for pipeline secrets
	secretPrefix = "jx-pipeline"
)

// kubeConfigSaver saves configs to Kubernetes secrets
type kubeConfigLoadSaver struct {
	client      kubernetes.Interface
	namespace   string
	serverKind  string
	serviceKind string
}

// NewKubeConfigSaver creates a new ConfigSaver that saves the configuration into Kubernetes secrets
func NewKubeConfigService(client kubernetes.Interface, namespace string, serverKind string, serviceKind string) ConfigService {
	ks := kubeConfigLoadSaver{
		client:      client,
		namespace:   namespace,
		serverKind:  serverKind,
		serviceKind: serviceKind,
	}
	return NewConfigService(&ks, &ks)
}

// LoadConfig loads the config from Kubernetes secrets
func (k *kubeConfigLoadSaver) LoadConfig() (*Config, error) {
	secrets, err := k.secrets()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving config from k8s secrets")
	}
	if secrets == nil {
		return nil, errors.New("No secrets found with config")
	}
	config := &Config{}
	for _, secret := range secrets.Items {
		labels := secret.Labels
		annotations := secret.Annotations
		if labels != nil && annotations != nil {
			url := annotations[annotationURL]
			name := annotations[annotationName]
			serviceKind := labels[labelServiceKind]
			if url != "" {
				user, err := k.userFromSecret(secret)
				if err != nil {
					continue
				}
				server := Server{
					URL:  url,
					Name: name,
					Kind: serviceKind,
					Users: []User{
						user,
					},
					CurrentUser: user.Username,
				}
				if config.Servers == nil {
					config.Servers = []Server{}
				}
				config.Servers = append(config.Servers, server)
				config.CurrentServer = server.URL
			}
		}
	}
	return config, nil
}

// SaveConfig saves the config to Kubernetes secrets. It will use one secret pre server configuration.
func (k *kubeConfigLoadSaver) SaveConfig(config *Config) error {
	for _, server := range config.Servers {
		name := k.secretName(server)
		labels := k.labels(server)
		annotations := k.annotations(server)
		secret, err := k.client.CoreV1().Secrets(k.namespace).Get(name, metav1.GetOptions{})
		create := false
		if err != nil {
			create = true
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      labels,
					Annotations: annotations,
				},
				Data: map[string][]byte{},
			}
		} else {
			secret.Labels = util.MergeMaps(secret.Labels, labels)
			secret.Annotations = util.MergeMaps(secret.Annotations, annotations)
		}
		user, err := server.GetCurrentUser()
		if err != nil {
			return errors.Wrap(err, "getting current user")
		}
		if user.Username == "" {
			return errors.New("empty username")
		}
		if user.ApiToken == "" && user.Password == "" {
			return errors.New("empty credentials")
		}
		secret.Data[usernameKey] = []byte(user.Username)
		if user.ApiToken != "" {
			secret.Data[passwordKey] = []byte(user.ApiToken)
		} else {
			secret.Data[passwordKey] = []byte(user.Password)
		}
		if create {
			if _, err := k.client.CoreV1().Secrets(k.namespace).Create(secret); err != nil {
				return errors.Wrapf(err, "creating secret %q", name)
			}
		} else {
			if _, err := k.client.CoreV1().Secrets(k.namespace).Update(secret); err != nil {
				return errors.Wrapf(err, "updating secret %q", name)
			}
		}
	}
	return nil
}

// secretName builds the secret name
func (k *kubeConfigLoadSaver) secretName(server Server) string {
	secretName := secretPrefix
	serverKind := strings.ToLower(k.serverKind)
	if serverKind != "" {
		secretName += "-" + serverKind
	}
	serviceKind := strings.ToLower(server.Kind)
	if serviceKind != "" {
		secretName += "-" + serviceKind
	}
	name := strings.ToLower(server.Name)
	if name != "" {
		secretName += "-" + name
	}
	return secretName
}

func (k *kubeConfigLoadSaver) labels(server Server) map[string]string {
	return map[string]string{
		labelCredentialsType: valueCredentialTypeUsernamePassword,
		labelCreatedBy:       valueCreatedByJX,
		labelKind:            k.serverKind,
		labelServiceKind:     server.Kind,
	}
}

func (k *kubeConfigLoadSaver) annotations(server Server) map[string]string {
	return map[string]string{
		annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server %s", server.URL),
		annotationURL:                    server.URL,
		annotationName:                   server.Name,
	}
}

func (k *kubeConfigLoadSaver) secrets() (*corev1.SecretList, error) {
	selector := labelKind + "=" + string(k.serverKind)
	if k.serviceKind != "" {
		selector = labelServiceKind + "=" + string(k.serviceKind)
	}
	opts := metav1.ListOptions{
		LabelSelector: selector,
	}
	return k.client.CoreV1().Secrets(k.namespace).List(opts)
}

func (k *kubeConfigLoadSaver) userFromSecret(secret corev1.Secret) (User, error) {
	data := secret.Data
	if data == nil {
		return User{}, fmt.Errorf("No user auth credentials found in secret '%s'", secret.Name)
	}
	username, ok := data[usernameKey]
	if !ok || len(username) == 0 {
		return User{}, fmt.Errorf("No usernmae found in secret '%s'", secret.Name)
	}
	password, ok := data[passwordKey]
	if !ok || len(password) == 0 {
		return User{}, fmt.Errorf("No password found in secret '%s'", secret.Name)
	}

	return User{
		Username: string(username),
		ApiToken: string(password),
	}, nil
}
