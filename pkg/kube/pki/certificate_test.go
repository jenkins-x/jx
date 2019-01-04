package pki_test

import (
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/kube/pki"
	certmng "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	certclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWaitCertificateIssuedReady(t *testing.T) {
	certclient := certclient.NewSimpleClientset()

	const name = "test"
	const ns = "test"
	cert := newCert(name, certmng.CertificateCondition{
		Type:   certmng.CertificateConditionReady,
		Status: certmng.ConditionTrue,
	})
	_, err := certclient.Certmanager().Certificates(ns).Create(cert)
	assert.NoError(t, err, "should create a test certificate whithout an error")

	err = pki.WaitCertificateIssuedReady(certclient, name, ns, 3*time.Second)
	assert.NoError(t, err, "should find a cert in ready state")

	err = certclient.Certmanager().Certificates(ns).Delete(name, &metav1.DeleteOptions{})
	assert.NoError(t, err, "should delete the test certificate whithout an error")

	cert = newCert(name, certmng.CertificateCondition{
		Type:   certmng.CertificateConditionValidationFailed,
		Status: certmng.ConditionTrue,
	})
	_, err = certclient.Certmanager().Certificates(ns).Create(cert)
	assert.NoError(t, err, "should create a test certificate whithout an error")

	err = pki.WaitCertificateIssuedReady(certclient, name, ns, 5*time.Second)
	assert.Error(t, err, "should not find a cert in ready state")
}

func newCert(name string, condition certmng.CertificateCondition) *certmng.Certificate {
	return &certmng.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: certmng.CertificateStatus{
			Conditions: []certmng.CertificateCondition{condition},
		},
	}
}
