package testhelpers

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = l.Close()
	}()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// WaitForPod waits for the specified duration for the given Pod to get into the 'Running' status.
func WaitForPod(pod *core_v1.Pod, namespace string, labels map[string]string, timeout time.Duration, kubeClient kubernetes.Interface) error {
	status := pod.Status
	watch, err := kubeClient.CoreV1().Pods(namespace).Watch(meta_v1.ListOptions{
		Watch:           true,
		ResourceVersion: pod.ResourceVersion,
		LabelSelector:   LabelSelector(labels),
	})
	if err != nil {
		return errors.Wrapf(err, "unable to create watch for pod '%s'", pod.Name)
	}

	func() {
		for {
			select {
			case events, ok := <-watch.ResultChan():
				if !ok {
					return
				}
				pod := events.Object.(*core_v1.Pod)
				log.Logger().Debugf("Pod status: %v", pod.Status.Phase)
				status = pod.Status
				if pod.Status.Phase != core_v1.PodPending {
					watch.Stop()
				}
			case <-time.After(timeout):
				log.Logger().Debugf("timeout to wait for pod active")
				watch.Stop()
			}
		}
	}()
	if status.Phase != core_v1.PodRunning {
		return errors.Errorf("Pod '%s' should be running", pod.Name)
	}
	return nil
}

// PortForward port forwards the container port of the specified pod in the given namespace to the specified local forwarding port.
// The functions returns a stop channel to stop port forwarding.
func PortForward(namespace string, podName string, containerPort string, forwardPort string, factory clients.Factory) (chan struct{}, error) {
	config, err := factory.CreateKubeConfig()
	if err != nil {
		return nil, err
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(config.Host, "https:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%s:%s", forwardPort, containerPort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
		}
		if len(errOut.String()) != 0 {
			log.Logger().Error(errOut.String())
		} else if len(out.String()) != 0 {
			log.Logger().Info(out.String())
		}
	}()

	go func() {
		err := forwarder.ForwardPorts()
		if err != nil {
			log.Logger().Errorf("error during port forwarding: %s", errOut.String())
		}
	}()

	return stopChan, err
}

// LabelSelector builds a Kubernetes label selector from the specified map.
func LabelSelector(labels map[string]string) string {
	selector := ""
	for k, v := range labels {
		selector = selector + fmt.Sprintf("%s=%s,", k, v)
	}
	selector = strings.TrimRight(selector, ",")
	return selector
}
