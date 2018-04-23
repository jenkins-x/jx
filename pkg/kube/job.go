package kube

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// waits for the job to complete
func WaitForJobToComplete(client *kubernetes.Clientset, namespace, jobName string, timeout time.Duration) error {

	job, err := client.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	selector, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return err
	}

	options := metav1.ListOptions{LabelSelector: selector.String()}

	w, err := client.BatchV1().Jobs(namespace).Watch(options)
	if err != nil {
		return err
	}

	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		job := event.Object.(*batchv1.Job)
		return IsJobComplete(job), nil
	}

	_, err = watch.Until(timeout, w, condition)
	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("job %s never completed", jobName)
	}
	return nil
}

// waits for the pods of a deployment to become ready
func DeleteJob(client *kubernetes.Clientset, namespace, name string) error {
	err := client.BatchV1().Jobs(namespace).Delete(name, metav1.NewDeleteOptions(0))
	if err != nil {
		return fmt.Errorf("error deleting job %s. error: %v", name, err)
	}
	return nil
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsJobComplete(job *batchv1.Job) bool {
	return IsJobCompleteConditionTrue(job.Status)
}

// IsPodReady retruns true if a pod is ready; false otherwise.
func IsJobCompleteConditionTrue(status batchv1.JobStatus) bool {
	condition := GetJobCompleteCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// Extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetJobCompleteCondition(status batchv1.JobStatus) *batchv1.JobCondition {
	_, condition := GetJobCondition(&status, batchv1.JobComplete)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetJobCondition(status *batchv1.JobStatus, conditionType batchv1.JobConditionType) (int, *batchv1.JobCondition) {
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
