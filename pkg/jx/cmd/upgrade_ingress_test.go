package cmd

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
)

type TestOptions struct {
	UpgradeIngressOptions
	Service v1.Service
}

func (o *TestOptions) Setup() {
	o.UpgradeIngressOptions = UpgradeIngressOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				kubeClient: testclient.NewSimpleClientset(),
			},
		},
		Issuer:           "letsencrypt-prod",
		TargetNamespaces: []string{"test"},
	}

	o.Service = v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: v1.ServiceSpec{},
	}

	o.Service.Annotations = map[string]string{}
	o.Service.Annotations[kube.ExposeAnnotation] = "true"
}

func TestAnnotateNoExisting(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.cleanServiceAnnotations()
	assert.NoError(t, err)

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "certmanager.k8s.io/issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestAnnotateWithExistingAnnotations(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	o.Service.Annotations[kube.ExposeIngressAnnotation] = "foo: bar\nkubernetes.io/ingress.class: nginx\nnginx.ingress.kubernetes.io/proxy-body-size: 500m"

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.cleanServiceAnnotations()
	assert.NoError(t, err)

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "foo: bar\nkubernetes.io/ingress.class: nginx\nnginx.ingress.kubernetes.io/proxy-body-size: 500m\ncertmanager.k8s.io/issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestAnnotateWithExistingCertManagerAnnotation(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	o.Service.Annotations[kube.ExposeIngressAnnotation] = kube.CertManagerAnnotation + ": letsencrypt-staging"

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.cleanServiceAnnotations()
	assert.NoError(t, err)

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "certmanager.k8s.io/issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestCleanExistingExposecontrollerReources(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	err := o.cleanExposecontrollerReources("test")
	assert.NoError(t, err)
}

func TestCleanServiceAnnotations(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	o.Service.Annotations[kube.ExposeURLAnnotation] = "http://foo.bar"

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.cleanServiceAnnotations()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})

	assert.Empty(t, rs.Annotations[kube.ExposeURLAnnotation])
	assert.NoError(t, err)
}
