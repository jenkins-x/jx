package kube

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// LogMasker replaces words in a log from a set of secrets
type LogMasker struct {
	ReplaceWords map[string]string
}

// LoadSecrets loads the secrets into the log masker
func (m *LogMasker) LoadSecrets(kubeClient kubernetes.Interface, ns string) error {
	resourceList, err := kubeClient.CoreV1().Secrets(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, secret := range resourceList.Items {
		m.LoadSecret(&secret)
	}
	return nil
}

// LoadSecret loads the secret data into the log masker
func (m *LogMasker) LoadSecret(secret *corev1.Secret) {
	if m.ReplaceWords == nil {
		m.ReplaceWords = map[string]string{}
	}

	if secret.Data != nil {
		for _, v := range secret.Data {
			if v != nil && len(v) > 0 {
				// key := string(k)
				value := string(v)

				m.ReplaceWords[value] = strings.Repeat("*", len(value))
			}
		}
	}
}

// MaskLog returns the text with all of the secrets masked out
func (m *LogMasker) MaskLog(text string) string {
	answer := text
	for k, v := range m.ReplaceWords {
		answer = strings.Replace(answer, k, v, -1)
	}
	return answer
}
