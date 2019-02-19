package builds

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetBuildPods returns all the build pods in the given namespace
func GetBuildPods(kubeClient kubernetes.Interface, ns string) ([]*corev1.Pod, error) {
	answer := []*corev1.Pod{}
	podList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: LabelBuildName,
	})
	if err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		podList, err = kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: LabelOldBuildName,
		})
		if err != nil {
			return nil, err
		}
	}
	pipelinePodList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: LabelPipelineRunName,
	})
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		copy := pod
		answer = append(answer, &copy)
	}
	for _, pod := range pipelinePodList.Items {
		copy := pod
		answer = append(answer, &copy)
	}
	return answer, nil
}
