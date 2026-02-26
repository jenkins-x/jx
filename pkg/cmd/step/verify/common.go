package verify

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *StepVerifyOptions) validateKaniko(ns string) error {
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	return kube.ValidateSecret(kubeClient, kube.SecretKaniko, kube.SecretKaniko, ns)
}

func (o *StepVerifyOptions) validateVelero(ns string) error {
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	return kube.ValidateSecret(kubeClient, kube.SecretVelero, "cloud", ns)
}

func (o *StepVerifyOptions) createSecret(ns string, name, key, data string) error {
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	err = kube.EnsureNamespaceCreated(kubeClient, ns, map[string]string{
		kube.LabelCreatedBy: "jx-boot",
	}, nil)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			key: []byte(data),
		},
	}

	_, err = kubeClient.CoreV1().Secrets(ns).Create(secret)
	if err != nil {
		return errors.Wrapf(err, "could not create the Secret %s in the namespace: %s", name, ns)
	}
	log.Logger().Infof("created Secret: %s in namespace: %s\n", util.ColorInfo(name), util.ColorInfo(ns))
	return nil

}
