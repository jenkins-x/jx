package iks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	ibmcloud "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
	"github.com/IBM-Cloud/bluemix-go/client"
	"github.com/IBM-Cloud/bluemix-go/helpers"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/IBM-Cloud/bluemix-go/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Clusters interface {
	GetClusterConfig(name string, target containerv1.ClusterTargetHeader) (string, error)
}
type clusters struct {
	*client.Client
}

// This name is the name known to cruiser
type ClusterConfig struct {
	ClusterID      string `json:"cluster_id"`
	ClusterName    string `json:"cluster_name"`
	ClusterType    string `json:"cluster_type"`
	ClusterPayTier string `json:"cluster_pay_tier"`
	Datacenter     string `json:"datacenter"`
	AccountID      string `json:"account_id"`
	Created        string `json:"created"`
}

func newClusterAPI(c *client.Client) Clusters {
	return &clusters{
		Client: c,
	}
}

func ComputeClusterConfigDir(name string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	resultDir := filepath.Join(usr.HomeDir, ".bluemix", "plugins", "container-service", "clusters", name)
	return resultDir, nil
}

func (r *clusters) GetClusterConfig(name string, target containerv1.ClusterTargetHeader) (string, error) {
	rawURL := fmt.Sprintf("/v1/clusters/%s/config", name)
	resultDir, err := ComputeClusterConfigDir(name)
	if err != nil {
		return "", fmt.Errorf("Error computing directory to download the cluster config")
	}

	err = os.MkdirAll(resultDir, 0755)
	if err != nil {
		return "", fmt.Errorf("Error creating directory to download the cluster config")
	}
	downloadPath := filepath.Join(resultDir, "config.zip")
	trace.Logger.Println("Will download the kubeconfig at", downloadPath)

	var out *os.File
	if out, err = os.Create(downloadPath); err != nil {
		return "", err
	}
	defer out.Close()                      //nolint:errcheck
	defer helpers.RemoveFile(downloadPath) //nolint:errcheck
	_, err = r.Client.Get(rawURL, out, target.ToMap())
	if err != nil {
		return "", err
	}
	trace.Logger.Println("Downloaded the kubeconfig at", downloadPath)
	if err = helpers.Unzip(downloadPath, resultDir); err != nil {
		return "", err
	}
	defer helpers.RemoveFilesWithPattern(resultDir, "[^(.yml)|(.pem)]$") //nolint:errcheck
	var kubedir, kubeyml string
	files, _ := ioutil.ReadDir(resultDir)
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "kube") {
			kubedir = filepath.Join(resultDir, f.Name())
			files, _ := ioutil.ReadDir(kubedir)
			for _, f := range files {
				old := filepath.Join(kubedir, f.Name())
				new := filepath.Join(kubedir, "../", f.Name())
				if strings.HasSuffix(f.Name(), ".yml") {
					kubeyml = new
				}
				err := os.Rename(old, new)
				if err != nil {
					return "", fmt.Errorf("Couldn't rename: %q", err)
				}
			}
			break
		}
	}
	if kubedir == "" {
		return "", errors.New("Unable to locate kube config in zip archive")
	}
	return filepath.Abs(kubeyml)
}

func GetClusterName() (string, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ConfigAccess().GetStartingConfig()
	if err != nil {
		return "", err
	}
	return config.Contexts[config.CurrentContext].Cluster, nil

}

func GetClusterID() (string, error) {

	// we can also get this from kubeconfig
	//token :=
	c := new(ibmcloud.Config)
	accountID, err := ConfigFromJSON(c)
	if err != nil {
		return "", err
	}

	s, err := session.New(c)
	if err != nil {
		return "", err
	}

	clusterAPI, err := containerv1.New(s)
	if err != nil {
		return "", err
	}

	clusterIF := clusterAPI.Clusters()
	clusterName, err := GetClusterName()
	if err != nil {
		return "", err
	}

	target := containerv1.ClusterTargetHeader{
		Region:    c.Region,
		AccountID: accountID,
	}

	clusterID, err := clusterIF.Find(clusterName, target)

	if err != nil {
		return "", err
	}
	return clusterID.ID, nil
}

func GetKubeClusterID(kubeClient kubernetes.Interface) (string, error) {

	clusterConfigCM, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get("cluster-info", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	clusterConfig := &ClusterConfig{}
	err = json.Unmarshal(clusterConfigCM.BinaryData["cluster-config.json"], clusterConfig)
	return clusterConfig.ClusterID, nil
}

func GetKubeClusterRegion(kubeClient kubernetes.Interface) (string, error) {

	clusterConfigCM, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get("crn-info-ibmc", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return clusterConfigCM.Data["CRN_REGION"], nil
}
