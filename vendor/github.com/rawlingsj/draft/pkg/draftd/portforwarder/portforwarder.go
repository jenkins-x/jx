package portforwarder

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/tiller/environment"
)

const (
	// DefaultDraftNamespace is the Kubernetes namespace in which the Draft pod runs by default.
	DefaultDraftNamespace string = environment.DefaultTillerNamespace
)

// New returns a tunnel to the Draft pod.
func New(clientset *kubernetes.Clientset, config *restclient.Config, namespace string) (*kube.Tunnel, error) {
	podName, err := getDraftPodName(clientset, namespace)
	if err != nil {
		return nil, err
	}
	const draftPort = 44135
	t := kube.NewTunnel(clientset.CoreV1().RESTClient(), config, namespace, podName, draftPort)
	return t, t.ForwardPort()
}

func getDraftPodName(clientset *kubernetes.Clientset, namespace string) (string, error) {
	// TODO use a const for labels
	selector := labels.Set{"app": "draft", "name": "draftd"}.AsSelector()
	pod, err := getFirstRunningPod(clientset, selector, namespace)
	if err != nil {
		return "", err
	}
	return pod.ObjectMeta.GetName(), nil
}

func getFirstRunningPod(clientset *kubernetes.Clientset, selector labels.Selector, namespace string) (*v1.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := clientset.CoreV1().Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find draftd")
	}
	for _, p := range pods.Items {
		if v1.IsPodReady(&p) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("could not find a ready draftd pod")
}
