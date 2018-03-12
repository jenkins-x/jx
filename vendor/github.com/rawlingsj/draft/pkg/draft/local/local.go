package local

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/helm/pkg/kube"

	"github.com/Azure/draft/pkg/draft/manifest"
)

// DraftLabelKey is the label selector key on a pod that allows
//  us to identify which draft app a pod is associated with
const DraftLabelKey = "draft"

type App struct {
	Name      string
	Namespace string
	Container string
}

type Connection struct {
	Tunnel    *kube.Tunnel
	PodName   string
	Clientset kubernetes.Interface
}

// DeployedApplication returns deployment information about the deployed instance
//  of the source code given a path to your draft.toml file and the name of the
//  draft environment
func DeployedApplication(draftTomlPath, draftEnvironment string) (*App, error) {
	var draftConfig manifest.Manifest
	if _, err := toml.DecodeFile(draftTomlPath, &draftConfig); err != nil {
		return nil, err
	}
	appConfig := draftConfig.Environments[draftEnvironment]

	return &App{Name: appConfig.Name, Namespace: appConfig.Namespace}, nil
}

// Connect creates a local tunnel to a Kubernetes pod running the application and returns the connection information
func (a *App) Connect(clientset kubernetes.Interface, clientConfig *restclient.Config) (*Connection, error) {
	tunnel, podName, err := a.NewTunnel(clientset, clientConfig)
	if err != nil {
		return nil, err
	}

	return &Connection{
		Tunnel:    tunnel,
		PodName:   podName,
		Clientset: clientset,
	}, nil
}

// NewTunnel creates and returns a tunnel to a Kubernetes pod running the application
func (a *App) NewTunnel(clientset kubernetes.Interface, config *restclient.Config) (*kube.Tunnel, string, error) {
	podName, containers, err := getAppPodNameAndContainers(a.Namespace, a.Name, clientset)
	if err != nil {
		return nil, "", err
	}

	port, err := getContainerPort(containers, a.Container)
	if err != nil {
		return nil, "", err
	}

	t := kube.NewTunnel(clientset.CoreV1().RESTClient(), config, a.Namespace, podName, port)
	if err != nil {
		return nil, "", err
	}

	return t, podName, t.ForwardPort()
}

func getContainerPort(containers []v1.Container, targetContainer string) (int, error) {
	var port int
	containerFound := false
	if targetContainer != "" {
		for _, container := range containers {
			if container.Name == targetContainer && !containerFound {
				containerFound = true
				port = int(container.Ports[0].ContainerPort)
			}
		}

		if containerFound == false {
			return 0, fmt.Errorf("container '%s' not found", targetContainer)
		}
	} else {
		// if not container is specified, default behavior is to
		//  grab first ContainerPort of first container
		port = int(containers[0].Ports[0].ContainerPort)
	}

	return port, nil

}

// RequestLogStream returns a stream of the application pod's logs
func (c *Connection) RequestLogStream(app *App, logLines int64) (io.ReadCloser, error) {

	req := c.Clientset.CoreV1().Pods(app.Namespace).GetLogs(c.PodName,
		&v1.PodLogOptions{
			Follow:    true,
			TailLines: &logLines,
			Container: app.Container,
		})

	return req.Stream()
}

func getAppPodNameAndContainers(namespace, labelVal string, clientset kubernetes.Interface) (string, []v1.Container, error) {
	selector := labels.Set{DraftLabelKey: labelVal}.AsSelector()
	pod, err := getFirstRunningPod(clientset, selector, namespace)
	if err != nil {
		return "", nil, err
	}
	return pod.ObjectMeta.GetName(), pod.Spec.Containers, nil
}

func getFirstRunningPod(clientset kubernetes.Interface, selector labels.Selector, namespace string) (*v1.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := clientset.CoreV1().Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("could not find ready pod")
	}
	for _, p := range pods.Items {
		if v1.IsPodReady(&p) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("could not find a ready pod")
}
