package kube

import (

	//v1 "k8s.io/kubernetes/pkg/api/v1"

	"k8s.io/api/core/v1"
)

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
