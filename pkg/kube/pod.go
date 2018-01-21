package kube

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"strings"
	"sort"
)

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// IsPodReady retruns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// Extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// waits for the pod to become ready using label selector to match the pod
func WaitForPodToBeReady(client *kubernetes.Clientset, selector labels.Selector, namespace string, timeout time.Duration) error {

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
		return fmt.Errorf("pod %s never became ready", selector)
	}
	return nil
}


func GetReadyPodNames(client *kubernetes.Clientset, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Pods %s", err)
	}
	for _, p := range list.Items {
		name := p.Name
		if (filter == "" || strings.Contains(name, filter) && IsPodReady(&p)) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}


func GetPodNames(client *kubernetes.Clientset, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Pods %s", err)
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
