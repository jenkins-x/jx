// package podutil exists for functions that exist in k8s.io/kubernetes but not in k8s.io/client-go. Most of the things here should be contributed upstream.

package podutil

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

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

// GetPod waits for a pod with the specified label to be ready, then returns it
// if no pod is ready, it checks every second until a pod is ready until timeout is reached
func GetPod(namespace string, draftLabelKey, name, annotationKey, buildID string, clientset kubernetes.Interface) (*v1.Pod, error) {
	var targetPod *v1.Pod
	s := newStopChan()

	listwatch := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", namespace, fields.Everything())
	_, controller := cache.NewInformer(listwatch, &v1.Pod{}, time.Second, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newPod := n.(*v1.Pod)

			// check the pod label and if pod is in terminating state
			if (newPod.Labels[draftLabelKey] != name) || (newPod.Annotations[annotationKey] != buildID) || (newPod.ObjectMeta.DeletionTimestamp != nil) {
				return
			}

			if IsPodReady(newPod) {
				targetPod = newPod
				s.closeOnce()
			}
		},
	})

	go func() {
		controller.Run(s.c)
	}()

	select {
	case <-s.c:
		return targetPod, nil
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("cannot get pod with buildID %v: timed out", buildID)
	}
}
