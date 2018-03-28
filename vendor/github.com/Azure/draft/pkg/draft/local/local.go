package local

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/Azure/draft/pkg/draft/tunnel"
	"github.com/Azure/draft/pkg/kube/podutil"
)

const (
	// DraftLabelKey is the label selector key on a pod that allows
	//  us to identify which draft app a pod is associated with
	DraftLabelKey = "draft"

	// BuildIDKey is the label selector key on a pod that specifies
	// the build ID of the application
	BuildIDKey = "buildID"
)

// App encapsulates information about an application to connect to
//
//  Name is the name of the application
//  Namespace is the Kubernetes namespace it is deployed in
//  Container is the name the name of the application container to connect to
type App struct {
	Name          string
	Namespace     string
	Container     string
	OverridePorts []string
}

// Connection encapsulated information to connect to an application
type Connection struct {
	ContainerConnections []*ContainerConnection
	PodName              string
	Clientset            kubernetes.Interface
}

// ContainerConnection encapsulates a connection to a container in a pod
type ContainerConnection struct {
	Tunnels       []*tunnel.Tunnel
	ContainerName string
}

// DeployedApplication returns deployment information about the deployed instance
//  of the source code given a path to your draft.toml file and the name of the
//  draft environment
func DeployedApplication(draftTomlPath, draftEnvironment string) (*App, error) {
	var draftConfig manifest.Manifest
	if _, err := toml.DecodeFile(draftTomlPath, &draftConfig); err != nil {
		return nil, err
	}

	appConfig, found := draftConfig.Environments[draftEnvironment]
	if !found {
		return nil, fmt.Errorf("Environment %v not found", draftEnvironment)
	}

	return &App{
		Name:          appConfig.Name,
		Namespace:     appConfig.Namespace,
		OverridePorts: appConfig.OverridePorts}, nil
}

// Connect tunnels to a Kubernetes pod running the application and returns the connection information
func (a *App) Connect(clientset kubernetes.Interface, clientConfig *restclient.Config, targetContainer string, overridePorts []string, buildID string) (*Connection, error) {
	var cc []*ContainerConnection

	pod, err := podutil.GetPod(a.Namespace, DraftLabelKey, a.Name, BuildIDKey, buildID, clientset)
	if err != nil {
		return nil, err
	}
	m, err := getPortMapping(overridePorts)
	if err != nil {
		return nil, err
	}

	// if no container was specified as flag, return tunnels to all containers in pod
	if targetContainer == "" {
		for _, c := range pod.Spec.Containers {
			var tt []*tunnel.Tunnel

			// iterate through all ports of the contaier and create tunnels
			for _, p := range c.Ports {
				remote := int(p.ContainerPort)
				local := m[remote]
				t := tunnel.NewWithLocalTunnel(clientset.CoreV1().RESTClient(), clientConfig, a.Namespace, pod.Name, remote, local)
				tt = append(tt, t)
			}
			cc = append(cc, &ContainerConnection{
				ContainerName: c.Name,
				Tunnels:       tt,
			})
		}

		return &Connection{
			ContainerConnections: cc,
			PodName:              pod.Name,
			Clientset:            clientset,
		}, nil
	}
	var tt []*tunnel.Tunnel

	// a container was specified - return tunnel to specified container
	ports, err := getTargetContainerPorts(pod.Spec.Containers, targetContainer)
	if err != nil {
		return nil, err
	}

	// iterate through all ports of the container and create tunnels
	for _, p := range ports {
		local := m[p]
		t := tunnel.NewWithLocalTunnel(clientset.CoreV1().RESTClient(), clientConfig, a.Namespace, pod.Name, p, local)
		tt = append(tt, t)
	}

	cc = append(cc, &ContainerConnection{
		ContainerName: targetContainer,
		Tunnels:       tt,
	})

	return &Connection{
		ContainerConnections: cc,
		PodName:              pod.Name,
		Clientset:            clientset,
	}, nil
}

func getPortMapping(overridePorts []string) (map[int]int, error) {
	var portMapping = make(map[int]int, len(overridePorts))

	for _, p := range overridePorts {
		m := strings.Split(p, ":")
		local, err := strconv.Atoi(m[0])
		if err != nil {
			return nil, fmt.Errorf("cannot get port mapping: %v", err)
		}

		remote, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("cannot get port mapping: %v", err)
		}

		// check if remote port already exists in port mapping
		_, exists := portMapping[remote]
		if exists {
			return nil, fmt.Errorf("remote port %v already mapped", remote)
		}

		// check if local port already exists in port mapping
		for _, l := range portMapping {
			if local == l {
				return nil, fmt.Errorf("local port %v already mapped", local)
			}
		}

		portMapping[remote] = local
	}

	return portMapping, nil
}

// RequestLogStream returns a stream of the application pod's logs
func (c *Connection) RequestLogStream(namespace string, containerName string, logLines int64) (io.ReadCloser, error) {
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(c.PodName,
		&v1.PodLogOptions{
			Follow:    true,
			TailLines: &logLines,
			Container: containerName,
		})

	return req.Stream()

}

func getTargetContainerPorts(containers []v1.Container, targetContainer string) ([]int, error) {
	var ports []int
	containerFound := false

	for _, c := range containers {

		if c.Name == targetContainer && !containerFound {
			containerFound = true
			for _, p := range c.Ports {
				ports = append(ports, int(p.ContainerPort))
			}
		}
	}

	if containerFound == false {
		return nil, fmt.Errorf("container '%s' not found", targetContainer)
	}

	return ports, nil
}
