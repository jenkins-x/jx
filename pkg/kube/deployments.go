package kube

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

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
