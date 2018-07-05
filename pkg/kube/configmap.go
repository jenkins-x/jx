package kube

import (
	"fmt"
	"strings"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	Exposecontroller = "exposecontroller"
)

func GetConfigmapData(client kubernetes.Interface, name, ns string) (map[string]string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s in namespace %s, %v", name, ns, err)
	}

	return cm.Data, nil
}

func GetCurrentDomain(client kubernetes.Interface, ns string) (string, error) {

	data, err := GetConfigmapData(client, Exposecontroller, ns)
	if err != nil {
		return "", err
	}
	return extractDomainValue(data)
}

func extractDomainValue(data map[string]string) (string, error) {

	//TODO change exposecontroller so it supports key/pair configmap as well as yaml file (for backwards compatibility)
	yaml := data["config.yml"]
	values := strings.Split(yaml, "\n")
	for _, line := range values {
		pair := strings.Split(line, ":")
		if pair[0] == "domain" {
			return strings.TrimSpace(pair[1]), nil
		}
	}

	return "", fmt.Errorf("failed to find a domain in %s configmap", Exposecontroller)
}
