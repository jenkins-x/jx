package kube

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetConfigmapData gets config map data
func GetConfigmapData(client kubernetes.Interface, name, ns string) (map[string]string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s in namespace %s, %v", name, ns, err)
	}

	return cm.Data, nil
}

// GetCurrentDomain gets the current domain
func GetCurrentDomain(client kubernetes.Interface, ns string) (string, error) {
	data, err := GetConfigmapData(client, ConfigMapIngressConfig, ns)
	if err != nil {
		data, err = GetConfigmapData(client, ConfigMapExposecontroller, ns)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to find ConfigMap in namespace %s for names %s and %s", ns, ConfigMapExposecontroller, ConfigMapIngressConfig)
		}
	}
	return ExtractDomainValue(data)
}

// ExtractDomainValue returns the domain value
func ExtractDomainValue(data map[string]string) (string, error) {
	answer := data["domain"]
	if answer != "" {
		return answer, nil
	}
	yaml := data["config.yml"]
	values := strings.Split(yaml, "\n")
	for _, line := range values {
		pair := strings.Split(line, ":")
		if pair[0] == "domain" {
			return strings.TrimSpace(pair[1]), nil
		}
	}
	return "", fmt.Errorf("failed to find a domain in %s configmap", ConfigMapExposecontroller)
}

// SaveAsConfigMap to the specified namespace as a config map.
func SaveAsConfigMap(c kubernetes.Interface, configMapName string, ns string, obj interface{}) (*v1.ConfigMap, error) {
	config := util.ToStringMapStringFromStruct(obj)

	cm, err := c.CoreV1().ConfigMaps(ns).Get(configMapName, meta_v1.GetOptions{})

	if err != nil {
		cm := &v1.ConfigMap{
			Data: config,
			ObjectMeta: meta_v1.ObjectMeta{
				Name: configMapName,
			},
		}
		_, err := c.CoreV1().ConfigMaps(ns).Create(cm)
		if err != nil {
			return &v1.ConfigMap{}, err
		}
		return cm, nil
	}

	// replace configmap values if it already exists
	cm.Data = config
	_, err = c.CoreV1().ConfigMaps(ns).Update(cm)
	if err != nil {
		return &v1.ConfigMap{}, err
	}
	return cm, nil
}
