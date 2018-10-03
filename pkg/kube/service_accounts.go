package kube

import (
	"encoding/json"

	"github.com/jenkins-x/jx/pkg/log"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type JsonPatch struct {
	ImagePullSecret *[]ImagePullSecret `json:"imagePullSecrets"`
}

type ImagePullSecret struct {
	Name string `json:"name"`
}

// PatchImagePullSecret patches the specified ImagePullSecret to the given service account
func PatchImagePullSecret(kubeClient kubernetes.Interface, ns string, sa string, imagePullSecret string) error {
	log.Infof("Namespace: %s\n", ns)
	log.Infof("Service Account: %s\n", sa)
	log.Infof("Secret: %s\n", imagePullSecret)

	// '{"imagePullSecrets": [{"name": "<secret>"}]}'
	ips := []ImagePullSecret{{
		Name: imagePullSecret,
	}}
	payload := JsonPatch{
		ImagePullSecret: &ips,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	log.Infof("Resultant JSON: %s\n", string(b))
	_, err = kubeClient.CoreV1().ServiceAccounts(ns).Patch(sa, types.StrategicMergePatchType, b)
	if err != nil {
		return err
	}
	return nil
}
