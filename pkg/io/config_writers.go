package io

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultWritePermissions = 0640
)

//ConfigWriter interface for writing auth configuration
type ConfigWriter interface {
	Write(config *auth.Config) error
}

//FileConfigWriter file config write which keeps the path to the configuration file
type FileConfigWriter struct {
	filename string
}

//NewFileConfigWriter creates a new file config writer
func NewFileConfigWriter(filename string) ConfigWriter {
	return &FileConfigWriter{
		filename: filename,
	}
}

//Write writes the auth configuration into a file
func (f *FileConfigWriter) Write(config *auth.Config) error {
	if f.filename == "" {
		return errors.New("No config file name defined")
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshaling the config to yaml")
	}
	err = ioutil.WriteFile(f.filename, content, defaultWritePermissions)
	return nil
}

const (
	usernameKey = "username"
	passwordKey = "password"
)

// KubeSecretsConfigWriter config writer into Kubernetes secrets
type KubeSecretsConfigWriter struct {
	client    kubernetes.Interface
	namespace string
}

// NewKubeSecretsConfigWriter creates a new Kubernetes secrets config writer
func NewKubeSecretsConfigWriter(client kubernetes.Interface, namespace string) ConfigWriter {
	return &KubeSecretsConfigWriter{
		client:    client,
		namespace: namespace,
	}
}

// Write write the config into Kuberntes secrets, it will one secret per server configuration
func (k *KubeSecretsConfigWriter) Write(config *auth.Config) error {
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
		user := server.PipelineUser()
		if err := user.Valid(); err != nil {
			return errors.Wrapf(err, "validating user '%s' of server '%s'", user.Username, server.Name)
		}
		secret.Data[usernameKey] = []byte(user.Username)
		secret.Data[passwordKey] = []byte(user.ApiToken)
		if create {
			_, err := k.client.CoreV1().Secrets(k.namespace).Create(secret)
			if err != nil {
				return errors.Wrapf(err, "creating secret '%s'", name)
			}
		} else {
			_, err := k.client.CoreV1().Secrets(k.namespace).Update(secret)
			if err != nil {
				return errors.Wrapf(err, "updating secret '%s'", name)
			}
		}
	}
	return nil
}

func (k *KubeSecretsConfigWriter) secretName(server *auth.Server) string {
	return kube.ToValidName(kube.SecretJenkinsPipelinePrefix + string(server.Kind) + "-" + string(server.ServiceKind) + "-" + server.Name)
}

func (k *KubeSecretsConfigWriter) labels(server *auth.Server) map[string]string {
	return map[string]string{
		kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
		kube.LabelCreatedBy:       kube.ValueCreatedByJX,
		kube.LabelKind:            string(server.Kind),
		kube.LabelServiceKind:     string(server.ServiceKind),
	}
}

func (k *KubeSecretsConfigWriter) annotations(server *auth.Server) map[string]string {
	return map[string]string{
		kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server %s", server.URL),
		kube.AnnotationURL:                    server.URL,
		kube.AnnotationName:                   server.Name,
	}
}
