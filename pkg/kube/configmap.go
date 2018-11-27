package kube

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetConfigmapData gets config map data
func GetConfigmapData(client kubernetes.Interface, name, ns string) (map[string]string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(name, metav1.GetOptions{})
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

	cm, err := c.CoreV1().ConfigMaps(ns).Get(configMapName, metav1.GetOptions{})

	if err != nil {
		cm := &v1.ConfigMap{
			Data: config,
			ObjectMeta: metav1.ObjectMeta{
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

// GetConfigMaps returns a map of the ConfigMaps along with a sorted list of names
func GetConfigMaps(kubeClient kubernetes.Interface, ns string) (map[string]*v1.ConfigMap, []string, error) {
	m := map[string]*v1.ConfigMap{}

	names := []string{}
	resourceList, err := kubeClient.CoreV1().ConfigMaps(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, names, err
	}
	for _, resource := range resourceList.Items {
		n := resource.Name
		copy := resource
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return m, names, nil
}

// DefaultModifyConfigMap default implementation of a function to modify
func DefaultModifyConfigMap(kubeClient kubernetes.Interface, ns string, name string, fn func(env *v1.ConfigMap) error, defaultConfigMap *v1.ConfigMap) (*v1.ConfigMap, error) {
	configMapInterface := kubeClient.CoreV1().ConfigMaps(ns)

	create := false
	configMap, err := configMapInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		create = true
		initialConfigMap := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Data: map[string]string{},
		}
		if defaultConfigMap != nil {
			initialConfigMap = *defaultConfigMap
		}
		configMap = &initialConfigMap
	}
	err = fn(configMap)
	if err != nil {
		return configMap, err
	}
	if create {
		_, err = configMapInterface.Create(configMap)
		if err != nil {
			return configMap, errors.Wrapf(err, "Failed to create ConfigMap %s in namespace %s", name, ns)
		}
		return configMap, err
	}
	_, err = configMapInterface.Update(configMap)
	if err != nil {
		return configMap, errors.Wrapf(err, "Failed to update ConfigMap %s in namespace %s", name, ns)
	}
	return configMap, nil
}
