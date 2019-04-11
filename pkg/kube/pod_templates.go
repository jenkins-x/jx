package kube

import (
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LoadPodTemplates loads the Jenkins pod templates from the given namepace
func LoadPodTemplates(kubeClient kubernetes.Interface, ns string) (map[string]*corev1.Pod, error) {
	answer := map[string]*corev1.Pod{}

	configMapName := ConfigMapJenkinsPodTemplates
	configMapInterface := kubeClient.CoreV1().ConfigMaps(ns)
	cm, err := configMapInterface.Get(configMapName, metav1.GetOptions{})
	if err == nil {
		for k, v := range cm.Data {
			pod := &corev1.Pod{}
			if v != "" {
				err := yaml.Unmarshal([]byte(v), pod)
				if err != nil {
					return answer, err
				}
				answer[k] = pod
			}
		}
		return answer, nil
	}

	// lets try load all the ConfigMaps by selector
	list, err2 := configMapInterface.List(metav1.ListOptions{
		LabelSelector: LabelKind + "=" + ValueKindPodTemplate,
	})
	if err2 != nil {
		return answer, util.CombineErrors(err, err2)
	}
	for _, cm := range list.Items {
		data := cm.Data
		found := false
		if data != nil {
			podYaml := data["pod"]
			if podYaml != "" {
				pod := &corev1.Pod{}
				err := yaml.Unmarshal([]byte(podYaml), pod)
				if err != nil {
					return answer, err
				}
				answer[cm.Name] = pod
				found = true
			}
		}
		if !found {
			logrus.Warnf("ConfigMap %s does not contain a pod key\n", cm.Name)
		}
	}
	return answer, nil
}
