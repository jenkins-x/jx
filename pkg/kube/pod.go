package kube

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	phase := pod.Status.Phase
	if phase != v1.PodRunning || pod.DeletionTimestamp != nil {
		return false
	}
	return IsPodReadyConditionTrue(pod.Status)
}

// credit https://github.com/kubernetes/kubernetes/blob/8719b4a/pkg/api/v1/pod/util.go
// IsPodReady retruns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

func PodStatus(pod *v1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}
	phase := pod.Status.Phase
	if IsPodReady(pod) {
		return "Ready"
	}
	return string(phase)
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
func WaitForPodToBeReady(client kubernetes.Interface, selector labels.Selector, namespace string, timeout time.Duration) error {
	options := meta_v1.ListOptions{LabelSelector: selector.String()}
	return waitForPodSelectorToBeReady(client, namespace, options, timeout)
}

func waitForPodSelectorToBeReady(client kubernetes.Interface, namespace string, options meta_v1.ListOptions, timeout time.Duration) error {
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
		return fmt.Errorf("pod %s never became ready", options.String())
	}
	return nil
}

// HasInitContainerStarted returns true if the given InitContainer has started running
func HasInitContainerStarted(pod *v1.Pod, idx int) bool {
	if pod == nil {
		return false
	}
	statuses := pod.Status.InitContainerStatuses
	if idx >= len(statuses) {
		return false
	}
	ic := statuses[idx]
	if ic.State.Running != nil || ic.State.Terminated != nil {
		return true
	}
	return false
}

// waits for the pod to become ready using the pod name
func WaitForPodNameToBeReady(client kubernetes.Interface, namespace string, name string, timeout time.Duration) error {
	options := meta_v1.ListOptions{
		// TODO
		//FieldSelector: fields.OneTermEqualSelector(api.ObjectNameField, name).String(),
		FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String(),
	}
	return waitForPodSelectorToBeReady(client, namespace, options, timeout)
}

func GetReadyPodNames(client kubernetes.Interface, ns string, filter string) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Pods %s", err)
	}
	for _, p := range list.Items {
		name := p.Name
		if filter == "" || strings.Contains(name, filter) && IsPodReady(&p) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func GetPodNames(client kubernetes.Interface, ns string, filter string) ([]string, error) {
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

func GetPods(client kubernetes.Interface, ns string, filter string) ([]string, map[string]*v1.Pod, error) {
	names := []string{}
	m := map[string]*v1.Pod{}
	list, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
	if err != nil {
		return names, m, fmt.Errorf("Failed to load Pods %s", err)
	}
	for _, d := range list.Items {
		c := d
		name := d.Name
		m[name] = &c
		if filter == "" || strings.Contains(name, filter) && d.DeletionTimestamp == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, m, nil
}

func GetPodsWithLabels(client kubernetes.Interface, ns string, selector string) ([]string, map[string]*v1.Pod, error) {
	names := []string{}
	m := map[string]*v1.Pod{}
	list, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return names, m, fmt.Errorf("Failed to load Pods %s", err)
	}
	for _, d := range list.Items {
		c := d
		name := d.Name
		m[name] = &c
		if d.DeletionTimestamp == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, m, nil
}

// GetDevPodNames returns the users dev pod names. If username is blank, all devpod names will be returned
func GetDevPodNames(client kubernetes.Interface, ns string, username string) ([]string, map[string]*v1.Pod, error) {
	names := []string{}
	m := map[string]*v1.Pod{}
	listOptions := meta_v1.ListOptions{}
	if username != "" {
		listOptions.LabelSelector = LabelDevPodUsername + "=" + username
	} else {
		listOptions.LabelSelector = LabelDevPodName
	}
	list, err := client.CoreV1().Pods(ns).List(listOptions)
	if err != nil {
		return names, m, fmt.Errorf("Failed to load Pods %s", err)
	}
	for _, d := range list.Items {
		c := d
		name := d.Name
		m[name] = &c
		names = append(names, name)
	}
	sort.Strings(names)
	return names, m, nil
}

// GetPodRestars returns the number of restarts of a POD
func GetPodRestarts(pod *v1.Pod) int32 {
	var restarts int32
	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return restarts
	}
	for _, status := range statuses {
		restarts += status.RestartCount
	}
	return restarts
}
