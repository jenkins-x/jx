package cmd

import (
	"encoding/json"
	"fmt"
	kh "github.com/Comcast/kuberhealthy/pkg/health"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

// KuberhealthyOptions options for the command
type KuberhealthyOptions struct {
	*opts.CommonOptions
}

var (
	kuberhealthyLong = templates.LongDesc(`
		Checks the current status of the Kubernetes cluster using https://github.com/Comcast/kuberhealthy

`)

	kuberhealthyExample = templates.Examples(`
		# checks the current health of the Kubernetes cluster
		jx kuberhealthy
`)
)

// NewCmdKuberhealthy creates the command
func NewCmdKuberhealthy(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &KuberhealthyOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "kuberhealthy",
		Short:   "kuberhealthy of the Kubernetes cluster using kuberhealthy",
		Long:    kuberhealthyLong,
		Example: kuberhealthyExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run runs this command
func (o *KuberhealthyOptions) Run() error {
	installed, err := o.checkKuberhealthyInstalled()
	if err != nil {
		return errors.Wrap(err, "failed to check if kuberhealthy is installed")
	}
	if !installed {
		return nil
	}

	URL, err := o.kuberhealthyURL()
	if err != nil {
		return errors.Wrap(err, "failed to get kuberhealthy URL")
	}

	state, err := o.kuberHealthyState(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get kuberhealthy state")
	}

	err = checkHealth(state)
	if err != nil {
		return errors.Wrap(err, "Your Kubernetes cluster is not healthy")
	}
	return nil
}

func (o *KuberhealthyOptions) checkKuberhealthyInstalled() (bool, error) {
	_, err := o.kuberhealthyService()
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Logger().Warnf("Kuberhealthy is not currently installed on the cluster")
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get kuberhealthy service")
	}
	return true, nil
}

func (o *KuberhealthyOptions) kuberhealthyURL() (string, error) {
	if o.InCluster() {
		khService, err := o.kuberhealthyService()
		if err != nil {
			return "", err
		}
		return khService.Spec.ClusterIP, nil
	}
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeClient")
	}
	ingressHost, err := kube.GetIngress(kubeClient, ns, "kuberhealthy")
	if err != nil {
		return "", errors.Wrap(err, "failed to get ingress")
	}
	return fmt.Sprintf("http://%s", ingressHost), nil
}

func (o *KuberhealthyOptions) kuberhealthyService() (*v1.Service, error) {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubeClient")
	}
	khService, err := kubeClient.CoreV1().Services(ns).Get("kuberhealthy", metav1.GetOptions{})
	return khService, err
}

func (o *KuberhealthyOptions) kuberHealthyState(kuberHealthIP string) (kh.State, error) {
	response, err := o.kuberHealthyRequest(kuberHealthIP)

	state := kh.State{}
	err = json.Unmarshal([]byte(response), &state)
	if err != nil {
		return state, errors.Wrapf(err, "failed to unmarshal to State")
	}
	return state, nil
}

func (o *KuberhealthyOptions) kuberHealthyRequest(kuberHealthURL string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", kuberHealthURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create request for %s", kuberHealthURL)
	}

	if !o.InCluster() {
		username, err := util.PickValue("Enter your admin username: ", "", true, "", o.In, o.Out, o.Err)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get username")
		}
		pwd, err := util.PickPassword("Enter your admin password:", "", o.In, o.Out, o.Err) // pragma: allowlist secret
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
		return nil, errors.Wrapf(err, "response code %b from hitting kuberhealthy URL %s", resp.StatusCode, kuberHealthURL)
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
