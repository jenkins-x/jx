package portforwarder

import (
	"fmt"

	"github.com/Azure/draft/pkg/kube/podutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/tiller/environment"
)

const (
	// DefaultDraftNamespace is the Kubernetes namespace in which the Draft pod runs by default.
	DefaultDraftNamespace string = environment.DefaultTillerNamespace
)

// New returns a tunnel to the Draft pod.
func New(client kubernetes.Interface, config *restclient.Config, namespace string) (*kube.Tunnel, error) {
	podName, err := getDraftPodName(client.CoreV1(), namespace)
	if err != nil {
		return nil, err
	}
	const draftPort = 44135
	t := kube.NewTunnel(client.Core().RESTClient(), config, namespace, podName, draftPort)
	return t, t.ForwardPort()
}

func getDraftPodName(client corev1.PodsGetter, namespace string) (string, error) {
	// TODO use a const for labels
	selector := labels.Set{"app": "draft", "name": "draftd"}.AsSelector()
	pod, err := getFirstRunningPod(client, selector, namespace)
	if err != nil {
		return "", err
	}
	return pod.ObjectMeta.GetName(), nil
}

func getFirstRunningPod(client corev1.PodsGetter, selector labels.Selector, namespace string) (*v1.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find draftd")
	}
	for _, p := range pods.Items {
		if podutil.IsPodReady(&p) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("could not find a ready draftd pod")
}
