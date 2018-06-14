package kube

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/api/core/v1"
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

func GetServices(client kubernetes.Interface, ns string) (map[string]*v1.Service, error) {
	answer := map[string]*v1.Service{}
	list, err := client.CoreV1().Services(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return answer, fmt.Errorf("failed to load Services %s", err)
	}
	for _, r := range list.Items {
		name := r.Name
		copy := r
		answer[name] = &copy
	}
	return answer, nil
}

func GetServiceNames(client kubernetes.Interface, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Services(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("failed to load Services %s", err)
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

func FindServiceURL(client kubernetes.Interface, namespace string, name string) (string, error) {
	svc, err := client.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}
	answer := GetServiceURL(svc)
	if answer != "" {
		return answer, nil
	}

	// lets try find the service via Ingress
	ing, err := client.ExtensionsV1beta1().Ingresses(namespace).Get(name, meta_v1.GetOptions{})
	if ing != nil && err == nil {
		if len(ing.Spec.Rules) > 0 {
			rule := ing.Spec.Rules[0]
			hostname := rule.Host
			for _, tls := range ing.Spec.TLS {
				for _, h := range tls.Hosts {
					if h != "" {
						return "https://" + h, nil
					}
				}
			}
			if hostname != "" {
				return "http://" + hostname, nil
			}
		}
	}
	return "", nil
}

// FindService looks up a service by name across all namespaces
func FindService(client kubernetes.Interface, name string) (*v1.Service, error) {
	nsl, err := client.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ns := range nsl.Items {
		svc, err := client.CoreV1().Services(ns.GetName()).Get(name, meta_v1.GetOptions{})
		if err == nil {
			return svc, nil
		}
	}
	return nil, errors.New("Service not found!")
}

func GetServiceURL(svc *v1.Service) string {
	url := ""
	if svc != nil && svc.Annotations != nil {
		url = svc.Annotations[ExposeURLAnnotation]
	}
	return url
}

func GetServiceURLFromName(c kubernetes.Interface, name, ns string) (string, error) {
	svc, err := c.CoreV1().Services(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return GetServiceURL(svc), nil
}

func FindServiceURLs(client kubernetes.Interface, namespace string) ([]ServiceURL, error) {
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
func WaitForExternalIP(client kubernetes.Interface, name, namespace string, timeout time.Duration) error {

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

func CreateServiceLink(client kubernetes.Interface, currentNamespace, targetNamespace, serviceName, externalURL string) error {
	annotations := make(map[string]string)
	annotations[ExposeURLAnnotation] = externalURL

	svc := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        serviceName,
			Namespace:   currentNamespace,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Type:         v1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, targetNamespace),
		},
	}

	_, err := client.CoreV1().Services(currentNamespace).Create(&svc)
	if err != nil {
		return err
	}

	return nil
}

func DeleteService(client *kubernetes.Clientset, namespace string, serviceName string) error {
	return client.CoreV1().Services(namespace).Delete(serviceName, &meta_v1.DeleteOptions{})
}

func GetService(client kubernetes.Interface, currentNamespace, targetNamespace, serviceName string) error {
	svc := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      serviceName,
			Namespace: currentNamespace,
		},
		Spec: v1.ServiceSpec{
			Type:         v1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, targetNamespace),
		},
	}
	_, err := client.CoreV1().Services(currentNamespace).Create(&svc)
	if err != nil {
		return err
	}
	return nil
}

func IsServicePresent(c kubernetes.Interface, name, ns string) (bool, error) {
	svc, err := c.CoreV1().Services(ns).Get(name, meta_v1.GetOptions{})
	if err != nil || svc == nil {
		return false, err
	}
	return true, nil
}
