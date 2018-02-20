package kube

import (
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const (
	ExposeURLAnnotation = "fabric8.io/exposeUrl"
)

type ServiceURL struct {
	Name string
	URL  string
}

func GetServices(client *kubernetes.Clientset, ns string) (map[string]*v1.Service, error) {
	answer := map[string]*v1.Service{}
	list, err := client.CoreV1().Services(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return answer, fmt.Errorf("Failed to load Services %s", err)
	}
	for _, r := range list.Items {
		name := r.Name
		copy := r
		answer[name] = &copy
	}
	return answer, nil
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

func GetServiceURLFromMap(services map[string]*v1.Service, name string) string {
	return GetServiceURL(services[name])
}

func FindServiceURL(client *kubernetes.Clientset, namespace string, name string) (string, error) {
	options := meta_v1.GetOptions{}
	svc, err := client.CoreV1().Services(namespace).Get(name, options)
	if err != nil {
		return "", err
	}
	return GetServiceURL(svc), nil
}

func GetServiceURL(svc *v1.Service) string {
	url := ""
	if svc != nil && svc.Annotations != nil {
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
		url := GetServiceURL(&svc)
		if len(url) > 0 {
			urls = append(urls, ServiceURL{
				Name: svc.Name,
				URL:  url,
			})
		}
	}
	return urls, nil
}

// waits for the pods of a deployment to become ready
func WaitForExternalIP(client *kubernetes.Clientset, name, namespace string, timeout time.Duration) error {

	options := meta_v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String(),
	}

	w, err := client.CoreV1().Services(namespace).Watch(options)

	if err != nil {
		return err
	}
	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		svc := event.Object.(*v1.Service)
		return HasExternalAddress(svc), nil
	}

	_, err = watch.Until(timeout, w, condition)
	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("service %s never became ready", name)
	}
	return nil
}

func HasExternalAddress(svc *v1.Service) bool {
	for _, v := range svc.Status.LoadBalancer.Ingress {
		if v.IP != "" || v.Hostname != "" {
			return true
		}
	}
	return false
}
