package kube

import (
	"fmt"
	"time"

	"context"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	tools_watch "k8s.io/client-go/tools/watch"
)

// waits for the job to complete
func WaitForJobToSucceeded(client kubernetes.Interface, namespace, jobName string, timeout time.Duration) error {
	job, err := client.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	options := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", job.Name).String()}

	w, err := client.BatchV1().Jobs(namespace).Watch(options)
	if err != nil {
		return err
	}

	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		job := event.Object.(*batchv1.Job)
		return job.Status.Succeeded == 1, nil
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	_, err = tools_watch.UntilWithoutRetry(ctx, w, condition)

	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("job %s never succeeded", jobName)
	}
	return nil
}

// waits for the job to terminate
func WaitForJobToTerminate(client kubernetes.Interface, namespace, jobName string, timeout time.Duration) error {
	job, err := client.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	options := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", job.Name).String()}

	w, err := client.BatchV1().Jobs(namespace).Watch(options)
	if err != nil {
		return err
	}

	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		job := event.Object.(*batchv1.Job)
		return job.Status.Succeeded == 1 || job.Status.Failed == 1, nil
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	_, err = tools_watch.UntilWithoutRetry(ctx, w, condition)
	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("job %s never terminated", jobName)
	}
	return nil
}

// IsJobSucceeded returns true if the job completed and did not fail
func IsJobSucceeded(job *batchv1.Job) bool {
	return IsJobFinished(job) && job.Status.Succeeded > 0
}

// IsJobFinished returns true if the job has completed
func IsJobFinished(job *batchv1.Job) bool {
	BackoffLimit := job.Spec.BackoffLimit
	return job.Status.CompletionTime != nil || (job.Status.Active == 0 && BackoffLimit != nil && job.Status.Failed >= *BackoffLimit)
}

func DeleteJob(client kubernetes.Interface, namespace, name string) error {
	err := client.BatchV1().Jobs(namespace).Delete(name, metav1.NewDeleteOptions(0))
	if err != nil {
		return fmt.Errorf("error deleting job %s. error: %v", name, err)
		return fmt.Errorf("job %s never succeeded", name)
	}
	return nil
}
