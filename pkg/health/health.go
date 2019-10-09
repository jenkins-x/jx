package health

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	kh "github.com/Comcast/kuberhealthy/pkg/health"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/cluster"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Kuberhealthy integrates and checks output from kuberhealthy
func Kuberhealthy(kubeClient kubernetes.Interface, namespace string) error {
	khService, err := kubeClient.CoreV1().Services(namespace).Get("kuberhealthy", metav1.GetOptions{})

	installed, err := checkKuberhealthyInstalled(err)
	if err != nil {
		return errors.Wrap(err, "failed to check if kuberhealthy is installed")
	}
	if !installed {
		return nil
	}

	URL, err := kuberhealthyURL(kubeClient, namespace, khService)
	if err != nil {
		return errors.Wrap(err, "failed to get kuberhealthy URL")
	}

	state, err := kuberHealthyState(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get kuberhealthy state")
	}

	err = checkHealth(state)
	if err != nil {
		return errors.Wrap(err, "Your Kubernetes cluster is not healthy")
	}
	return nil
}

func checkKuberhealthyInstalled(err error) (bool, error) {
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Logger().Warnf("Kuberhealthy (https://github.com/Comcast/kuberhealthy) " +
				"is not currently installed on the cluster")
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get kuberhealthy service")
	}
	return true, nil
}

func kuberhealthyURL(kubeClient kubernetes.Interface, namespace string, khService *v1.Service) (string, error) {
	if cluster.IsInCluster() {
		return khService.Spec.ClusterIP, nil
	}
	ingressHost, err := kube.GetIngress(kubeClient, namespace, "kuberhealthy")
	if err != nil {
		return "", errors.Wrap(err, "failed to get ingress")
	}
	return fmt.Sprintf("http://%s", ingressHost), nil
}

func kuberHealthyState(kuberHealthIP string) (kh.State, error) {
	state := kh.State{}
	response, err := kuberHealthyRequest(kuberHealthIP)
	if err != nil {
		return state, errors.Wrapf(err, "failed to get response from kuberhealthy")
	}

	err = json.Unmarshal(response, &state)
	if err != nil {
		return state, errors.Wrapf(err, "failed to unmarshal to State")
	}
	return state, nil
}

func kuberHealthyRequest(kuberHealthURL string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", kuberHealthURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create request for %s", kuberHealthURL)
	}

	if !cluster.IsInCluster() {
		username, err := util.PickValue("Enter your admin username: ", "", true, "", os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get username")
		}
		pwd, err := util.PickPassword("Enter your admin password:", "", os.Stdin, os.Stdout, os.Stderr) // pragma: allowlist secret
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get password")
		}
		req.SetBasicAuth(username, pwd)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send request for %s", kuberHealthURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response code %d from kuberhealthy URL %s", resp.StatusCode, kuberHealthURL)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "reading response body %s", resp.Body)
	}
	return b, nil
}

func checkHealth(state kh.State) error {
	if state.OK != true {
		failures := make(map[string]kh.CheckDetails)
		for k, check := range state.CheckDetails {
			if check.OK != true {
				failures[k] = check
			}
		}
		jsonString, err := json.Marshal(failures)
		if err == nil {
			log.Logger().Infof("failures: %v", string(jsonString))
			return errors.New(string(jsonString))
		}
	}
	log.Logger().Infof("Your Kubernetes cluster is %s", util.ColorInfo("HEALTHY"))
	return nil
}
