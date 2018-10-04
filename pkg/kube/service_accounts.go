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

// PatchImagePullSecrets patches the specified ImagePullSecrets to the given service account
func PatchImagePullSecrets(kubeClient kubernetes.Interface, ns string, sa string, imagePullSecrets []string) error {
	log.Infof("Namespace: %s\n", ns)
	log.Infof("Service Account: %s\n", sa)
	log.Infof("Secret: %s\n", imagePullSecrets)

	// '{"imagePullSecrets": [{"name": "<secret>"}]}'
	var ips []ImagePullSecret
	for _, secret := range imagePullSecrets {
		jsonSecret := ImagePullSecret{
			Name: secret,
		}
		ips = append(ips, jsonSecret)
	}
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
