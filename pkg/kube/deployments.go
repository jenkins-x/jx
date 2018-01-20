package kube

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func GetDeploymentNames(client *kubernetes.Clientset, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.AppsV1beta2().Deployments(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Deployments %s", err)
	}
	for _, d := range list.Items {
		name := d.Name
		if filter == "" || strings.Contains(name, filter) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func IsDeploymentRunning(client *kubernetes.Clientset, name, namespace string) (bool, error) {
	options := meta_v1.GetOptions{}

	d, err := client.ExtensionsV1beta1().Deployments(namespace).Get(name, options)
	if err != nil {
		return false, err
	}

	if d.Status.ReadyReplicas > 0 {
		return true, nil
	}
	return false, nil
}

func WaitForAllDeploymentsToBeReady(client *kubernetes.Clientset, namespace string, timeoutPerDeploy time.Duration) error {
	deployList, err := client.ExtensionsV1beta1().Deployments(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	if deployList == nil || len(deployList.Items) == 0 {
		return fmt.Errorf("no deployments found in namespace %s", namespace)
	}

	for _, d := range deployList.Items {
		err = WaitForDeploymentToBeReady(client, d.Name, namespace, timeoutPerDeploy)
		if err != nil {
			log.Warnf("deployment %s failed to become ready in namespase %s", d.Name, namespace)
		}
	}
	return nil
}

// waits for the pods of a deployment to become ready
func WaitForDeploymentToBeReady(client *kubernetes.Clientset, name, namespace string, timeout time.Duration) error {

	d, err := client.ExtensionsV1beta1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}

	selector, err := meta_v1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return err
	}

	options := meta_v1.ListOptions{LabelSelector: selector.String()}

	w, err := client.CoreV1().Pods(namespace).Watch(options)

	if err != nil {
		return err
	}
	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		pod := event.Object.(*v1.Pod)
		return IsPodReady(pod), nil
	}

	_, err = watch.Until(timeout, w, condition)
	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("deployment %s never became ready", name)
	}
	return nil
}
