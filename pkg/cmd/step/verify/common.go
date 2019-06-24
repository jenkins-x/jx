package verify

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *StepVerifyOptions) validateKaniko(ns string) error {
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(kube.SecretKaniko, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not find the Secret %s in the namespace: %s", kube.SecretKaniko, ns)
	}
	if secret.Data == nil || len(secret.Data[kube.SecretKaniko]) == 0 {
		return fmt.Errorf("the Secret %s in the namespace: %s does not have a key: %s", kube.SecretKaniko, ns, kube.SecretKaniko)
	}
	log.Logger().Infof("kaniko is valid: there is a Secret: %s in namespace: %s\n", util.ColorInfo(kube.SecretKaniko), util.ColorInfo(ns))
	return nil
}

func (o *StepVerifyOptions) createKanikoSecret(ns string, data string) error {
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	name := kube.SecretKaniko
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			name: []byte(data),
		},
	}

	_, err = kubeClient.CoreV1().Secrets(ns).Create(secret)
	if err != nil {
		return errors.Wrapf(err, "could not create the Secret %s in the namespace: %s", name, ns)
	}
	log.Logger().Infof("created kaniko Secret: %s in namespace: %s\n", util.ColorInfo(name), util.ColorInfo(ns))
	return nil

}
