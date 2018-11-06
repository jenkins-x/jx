package io

import (
	"fmt"
	"io/ioutil"

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
	Read() (*auth.AuthConfig, error)
}

//FileConfigReader keeps the path to the configration file
type FileConfigReader struct {
	filename string
}

//NewFileConfigReader creates a new file config reader
func NewFileConfigReader(filename string) *FileConfigReader {
	return &FileConfigReader{
		filename: filename,
	}
}

//Read reads the configuration from a file
func (f *FileConfigReader) Read() (*auth.AuthConfig, error) {
	config := &auth.AuthConfig{}
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
type ServerRetrieverFn func() (name string, url string, kind string)

//EventConfigReader keeps the prefix of the env variables where the user auth config is stored
// and also a server config retriever
type EnvConfigReader struct {
	prefix          string
	serverRetriever ServerRetrieverFn
}

//NewEnvConfigReader creates a new environment config reader
func NewEnvConfigReader(envPrefix string, serverRetriever ServerRetrieverFn) *EnvConfigReader {
	return &EnvConfigReader{
		prefix:          envPrefix,
		serverRetriever: serverRetriever,
	}
}

//Read reads the configuration from environment
func (e *EnvConfigReader) Read() (*auth.AuthConfig, error) {
	if e.serverRetriever == nil {
		return nil, errors.New("No server retriever function provider in env config reader")
	}
	config := &auth.AuthConfig{}
	userAuth := auth.CreateAuthUserFromEnvironment(e.prefix)
	if userAuth.IsInvalid() {
		return nil, errors.New("Invalid user found in the environment variables")
	}
	servername, url, kind := e.serverRetriever()
	config.Servers = []*auth.AuthServer{{
		Name:  servername,
		URL:   url,
		Kind:  kind,
		Users: []*auth.UserAuth{&userAuth},
	}}
	return config, nil
}

//NewKubeSecretsConfigReader config reader for Kubernetes secrets
type KubeSecretsConfigReader struct {
	client      kubernetes.Interface
	namespace   string
	kind        string
	serviceKind string
}

//NewKubeSecretsConfigReader creates a new Kubernetes config reader
func NewKubeSecretsConfigReader(client kubernetes.Interface, namespace string,
	kind string, serviceKind string) *KubeSecretsConfigReader {
	return &KubeSecretsConfigReader{
		client:      client,
		namespace:   namespace,
		kind:        kind,
		serviceKind: serviceKind,
	}
}

//Read reads the config from Kuberntes secrets
func (k *KubeSecretsConfigReader) Read() (*auth.AuthConfig, error) {
	secrets, err := k.secrets()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving config from k8s secrets")
	}
	if secrets == nil {
		return nil, errors.New("No secrets found with config")
	}
	config := &auth.AuthConfig{}
	for _, secret := range secrets.Items {
		labels := secret.Labels
		annotations := secret.Annotations
		if labels != nil && annotations != nil {
			url := annotations[kube.AnnotationURL]
			name := annotations[kube.AnnotationName]
			serviceKind := labels[kube.LabelServiceKind]
			if url != "" {
				user, err := k.userAuthFromSecret(secret)
				if err != nil {
					continue
				}
				if user.IsInvalid() {
					continue
				}
				server := &auth.AuthServer{
					URL:  url,
					Name: name,
					Kind: serviceKind,
					Users: []*auth.UserAuth{
						user,
					},
				}
				config.AddServer(server)
			}
		}
	}
	return config, nil
}

func (k *KubeSecretsConfigReader) userAuthFromSecret(secret corev1.Secret) (*auth.UserAuth, error) {
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

	return &auth.UserAuth{
		Username: string(username),
		ApiToken: string(password),
	}, nil
}

func (k *KubeSecretsConfigReader) secrets() (*corev1.SecretList, error) {
	selector := kube.LabelKind + "=" + k.kind
	if k.serviceKind != "" {
		selector = kube.LabelServiceKind + "=" + k.serviceKind
	}
	opts := metav1.ListOptions{
		LabelSelector: selector,
	}
	return k.client.CoreV1().Secrets(k.namespace).List(opts)
}
