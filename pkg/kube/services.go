package kube

import (
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"fmt"
	"strings"
	"sort"
)

const (
	ExposeURLAnnotation = "fabric8.io/exposeUrl"
)

type ServiceURL struct {
	Name string
	URL  string
}

func GetServiceNames(client *kubernetes.Clientset, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Services(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Services %s", err)
	}
	for _, r := range list.Items {
		name := r.Name
		if filter == "" || strings.Contains(name, filter) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func FindServiceURL(client *kubernetes.Clientset, namespace string, name string) (string, error) {
	options := meta_v1.GetOptions{}
	svc, err := client.CoreV1().Services(namespace).Get(name, options)
	if err != nil {
		return "", err
	}
	return getServiceURL(svc), nil
}

func getServiceURL(svc *v1.Service) string {
	url := ""
	if svc.Annotations != nil {
		url = svc.Annotations[ExposeURLAnnotation]
	}
	return url
}

func FindServiceURLs(client *kubernetes.Clientset, namespace string) ([]ServiceURL, error) {
	options := meta_v1.ListOptions{}
	urls := []ServiceURL{}
	svcs, err := client.CoreV1().Services(namespace).List(options)
	if err != nil {
		return urls, err
	}
	for _, svc := range svcs.Items {
		url := getServiceURL(&svc)
		if len(url) > 0 {
			urls = append(urls, ServiceURL{
				Name: svc.Name,
				URL:  url,
			})
		}
	}
	return urls, nil
}
