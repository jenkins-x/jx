package verify

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *StepVerifyOptions) validateKaniko() error {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
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
