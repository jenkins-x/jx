package pki

import (
	"time"

	certmng "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	certclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitCertificateIssuedReady wait for a certificate issued by cert-manager until is ready or the timeout is reached
func WaitCertificateIssuedReady(client certclient.Interface, name string, ns string, timeout time.Duration) error {
	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		cert, err := client.Certmanager().Certificates(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			logrus.Warnf("Failed getting certificate %q: %v", name, err)
			return false, nil
		}
		isReady := cert.HasCondition(certmng.CertificateCondition{
			Type:   certmng.CertificateConditionReady,
			Status: certmng.ConditionTrue,
		})
		if !isReady {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "waiting for certificate %q to be ready in namespace %q", name, ns)
	}
	return nil
}
