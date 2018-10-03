package kube

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"path/filepath"
)

const extensionsConfigDefaultFile = "jenkins-x-extensions.yaml"
const extensionsConfigDefaultConfigMap = "jenkins-x-extensions"

type ExtensionsConfig struct {
	Extensions map[string]ExtensionConfig `json:"extensions"`
}

type ExtensionConfig struct {
	Parameters map[string]string `json: "parameters"`
}

func (extensionsConfig *ExtensionsConfig) LoadFromFile() (cfg *ExtensionsConfig, err error) {
	extensionsYamlPath, err := filepath.Abs(extensionsConfigDefaultFile)
	if err != nil {
		return nil, err
	}
	extensionsYaml, err := ioutil.ReadFile(extensionsYamlPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(extensionsYaml, extensionsConfig)
	if err != nil {
		return nil, err
	}
	return extensionsConfig, nil
}

func (extensionsConfig *ExtensionsConfig) LoadFromConfigMap(client kubernetes.Interface, namespace string) (cfg *ExtensionsConfig, err error) {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(extensionsConfigDefaultConfigMap, metav1.GetOptions{})
	if err != nil {
		// CM doesn't exist, create it
		cm, err = client.CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: extensionsConfigDefaultConfigMap,
			},
		})
		if err != nil {
			return nil, err
		}
	}
	extensionsConfig.Extensions = make(map[string]ExtensionConfig)
	for k, v := range cm.Data {
		parameters := make(map[string]string, 0)
		err = yaml.Unmarshal([]byte(v), &parameters)
		if err != nil {
			return nil, err
		}
		extensionsConfig.Extensions[k] = ExtensionConfig{
			Parameters: parameters,
		}
	}
	return extensionsConfig, nil
}
