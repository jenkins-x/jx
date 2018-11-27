package certmanager

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/client-go/kubernetes"
)

// CopyCertmanagerResources copies certmanager resources to the targetNamespace
func CopyCertmanagerResources(targetNamespace string, ic kube.IngressConfig, kubeClient kubernetes.Interface) error {
	if ic.TLS {
		err := kube.CleanCertmanagerResources(kubeClient, targetNamespace, ic)
		if err != nil {
			return fmt.Errorf("failed to create certmanager resources in target namespace %s: %v", targetNamespace, err)
		}
	}

	return nil
}
