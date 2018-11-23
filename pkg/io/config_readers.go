package io

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//ConfigReader interface for reading auth configuration
type ConfigReader interface {
	Read() (*auth.Config, error)
}

//FileConfigReader keeps the path to the configration file
type FileConfigReader struct {
	filename string
}

//NewFileConfigReader creates a new file config reader
func NewFileConfigReader(filename string) ConfigReader {
	return &FileConfigReader{
		filename: filename,
	}
}

//Read reads the configuration from a file
func (f *FileConfigReader) Read() (*auth.Config, error) {
	config := &auth.Config{}
	if f.filename == "" {
		return nil, errors.New("No config file name defined")
	}
	exists, err := util.FileExists(f.filename)
	if err != nil {
		return nil, errors.Wrapf(err, "checking if the file config file '%s' exits", f.filename)
	}
	if !exists {
		return nil, fmt.Errorf("Config file '%s' does not exist", f.filename)
	}
	content, err := ioutil.ReadFile(f.filename)
	if err != nil {
		return nil, errors.Wrapf(err, "reading the content of config file '%s'", f.filename)
	}
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshaling the content of config file '%s'", f.filename)
	}
	return config, nil
}

//ServerRetrieverFn retrives the server config
type ServerRetrieverFn func() (name string, url string,
	kind auth.ServerKind, serviceKind auth.ServiceKind)

//EnvConfigReader keeps the prefix of the env variables where the user auth config is stored
// and also a server config retriever
type EnvConfigReader struct {
	prefix          string
	serverRetriever ServerRetrieverFn
}

const (
	usernameSuffix    = "_USERNAME"
	apiTokenSuffix    = "_API_TOKEN"
	bearerTokenSuffix = "_BEARER_TOKEN"
	defaultUsername   = "dummy"
)

// UsernameEnv builds the username environment variable name
func usernameEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + usernameSuffix
}

// ApiTokenEnv builds the api token environment variable name
func apiTokenEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + apiTokenSuffix
}

// BearerTokenEnv builds the bearer token environment variable name
func bearerTokenEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + bearerTokenSuffix
}

//NewEnvConfigReader creates a new environment config reader
func NewEnvConfigReader(envPrefix string, serverRetriever ServerRetrieverFn) ConfigReader {
	return &EnvConfigReader{
		prefix:          envPrefix,
		serverRetriever: serverRetriever,
	}
}

//Read reads the configuration from environment
func (e *EnvConfigReader) Read() (*auth.Config, error) {
	if e.serverRetriever == nil {
		return nil, errors.New("No server retriever function provider in env config reader")
	}
	config := &auth.Config{}
	user := e.userFromEnv(e.prefix)
	if err := user.Valid(); err != nil {
		return nil, errors.Wrap(err, "validating user from environment")
	}
	servername, url, kind, serviceKind := e.serverRetriever()
	config.Servers = []*auth.Server{{
		Name:        servername,
		URL:         url,
		Kind:        kind,
		ServiceKind: serviceKind,
		Users:       []*auth.User{&user},
	}}
	return config, nil
}

func (e *EnvConfigReader) userFromEnv(prefix string) auth.User {
	user := auth.User{
		Kind: auth.UserKindPipeline,
	}
	username, set := os.LookupEnv(usernameEnv(prefix))
	if set {
		user.Username = username
	}
	apiToken, set := os.LookupEnv(apiTokenEnv(prefix))
	if set {
		user.ApiToken = apiToken
	}
	bearerToken, set := os.LookupEnv(bearerTokenEnv(prefix))
	if set {
		user.BearerToken = bearerToken
	}

	if user.ApiToken != "" || user.Password != "" {
		if user.Username == "" {
			user.Username = defaultUsername
		}
	}
	return user
}

//KubeSecretsConfigReader config reader for Kubernetes secrets
type KubeSecretsConfigReader struct {
	client      kubernetes.Interface
	namespace   string
	serverKind  auth.ServerKind
	serviceKind auth.ServiceKind
}

//NewKubeSecretsConfigReader creates a new Kubernetes config reader
func NewKubeSecretsConfigReader(client kubernetes.Interface, namespace string,
	serverKind auth.ServerKind, serviceKind auth.ServiceKind) ConfigReader {
	return &KubeSecretsConfigReader{
		client:      client,
		namespace:   namespace,
		serverKind:  serverKind,
		serviceKind: serviceKind,
	}
}

//Read reads the config from Kuberntes secrets
func (k *KubeSecretsConfigReader) Read() (*auth.Config, error) {
	secrets, err := k.secrets()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving config from k8s secrets")
	}
	if secrets == nil {
		return nil, errors.New("No secrets found with config")
	}
	config := &auth.Config{}
	for _, secret := range secrets.Items {
		labels := secret.Labels
		annotations := secret.Annotations
		if labels != nil && annotations != nil {
			url := annotations[kube.AnnotationURL]
			name := annotations[kube.AnnotationName]
			serverKind := auth.ServerKind(labels[kube.LabelKind])
			serviceKind := auth.ServiceKind(labels[kube.LabelServiceKind])
			if url != "" {
				user, err := k.userFromSecret(secret)
				if err != nil {
					continue
				}
				if err := user.Valid(); err != nil {
					continue
				}
				server := &auth.Server{
					URL:         url,
					Name:        name,
					Kind:        serverKind,
					ServiceKind: serviceKind,
					Users: []*auth.User{
						user,
					},
				}
				if config.Servers == nil {
					config.Servers = []*auth.Server{}
				}
				config.Servers = append(config.Servers, server)
			}
		}
	}
	return config, nil
}

func (k *KubeSecretsConfigReader) userFromSecret(secret corev1.Secret) (*auth.User, error) {
	data := secret.Data
	if data == nil {
		return nil, fmt.Errorf("No user auth credentials found in secret '%s'", secret.Name)
	}
	username, ok := data[kube.SecretDataUsername]
	if !ok || len(username) == 0 {
		return nil, fmt.Errorf("No usernmae found in secret '%s'", secret.Name)
	}
	password, ok := data[kube.SecretDataPassword]
	if !ok || len(password) == 0 {
		return nil, fmt.Errorf("No password found in secret '%s'", secret.Name)
	}

	return &auth.User{
		Username: string(username),
		ApiToken: string(password),
		Kind:     auth.UserKindPipeline,
	}, nil
}

func (k *KubeSecretsConfigReader) secrets() (*corev1.SecretList, error) {
	selector := kube.LabelKind + "=" + string(k.serverKind)
	if k.serviceKind != "" {
		selector = kube.LabelServiceKind + "=" + string(k.serviceKind)
	}
	opts := metav1.ListOptions{
		LabelSelector: selector,
	}
	return k.client.CoreV1().Secrets(k.namespace).List(opts)
}
