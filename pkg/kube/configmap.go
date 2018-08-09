package kube

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetConfigmapData(client kubernetes.Interface, name, ns string) (map[string]string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s in namespace %s, %v", name, ns, err)
	}

	return cm.Data, nil
}

func GetCurrentDomain(client kubernetes.Interface, ns string) (string, error) {
	data, err := GetConfigmapData(client, ConfigMapIngressConfig, ns)
	if err != nil {
		data, err = GetConfigmapData(client, ConfigMapExposecontroller, ns)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to find ConfigMap in namespace %s for names %s and %s", ns, ConfigMapExposecontroller, ConfigMapIngressConfig)
		}
	}
	return extractDomainValue(data)
}

func extractDomainValue(data map[string]string) (string, error) {
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
