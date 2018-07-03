package cmd

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//testclient "k8s.io/client-go/kubernetes/fake"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jx/cmd/certmanager"
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
				//kubeClient: testclient.NewSimpleClientset(),
			},
		},
		ClusterIssuer:    "letsencrypt-prod",
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

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "certmanager.k8s.io/cluster-issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestAnnotateWithExistingAnnotation(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	o.Service.Annotations[kube.ExposeIngressAnnotation] = "foo: bar"

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "foo: bar\ncertmanager.k8s.io/cluster-issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestAnnotateWithExistingCertManagerAnnotation(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	o.Service.Annotations[kube.ExposeIngressAnnotation] = "foo: bar\n" + kube.CertManagerAnnotation + ": letsencrypt-foo"

	_, err := o.kubeClient.CoreV1().Services("test").Create(&o.Service)
	assert.NoError(t, err)

	err = o.annotateExposedServicesWithCertManager()
	assert.NoError(t, err)

	rs, err := o.kubeClient.CoreV1().Services("test").Get("foo", metav1.GetOptions{})
	ingressAnnotations := rs.Annotations[kube.ExposeIngressAnnotation]

	assert.Equal(t, "foo: bar\ncertmanager.k8s.io/cluster-issuer: letsencrypt-prod", ingressAnnotations)
	assert.NoError(t, err)
}

func TestCleanExistingExposecontrollerReources(t *testing.T) {

	o := TestOptions{}
	o.Setup()

	err := o.cleanExposecontrollerReources("test")
	assert.NoError(t, err)
}

func TestMe(t *testing.T) {

	o := TestOptions{}
	o.Setup()
	o.Factory = NewFactory()
	//c, err := o.KubeRESTClient()

	c, _, err := o.KubeClient()

	assert.NoError(t, err)
	assert.NotEmpty(t, c)
	//issuerProd := fmt.Sprintf(certmanager.Cert_manager_issuer_prod, "rawlingsj80@gmail.com")
	//j2, err := yaml.YAMLToJSON([]byte(issuerProd))
	//
	//resp, err := c.CoreV1().RESTClient().Get().RequestURI("/apis/certmanager.k8s.io/v1alpha1/clusterissuers").Body(j2).DoRaw()
	//assert.NoError(t, err)

	//resp, err := o.kubeClient.CoreV1().RESTClient().Get().RequestURI("/apis/certmanager.k8s.io/v1alpha1/clusterissuers").Name("letsencrypt-staging").DoRaw()

	cert := fmt.Sprintf(certmanager.Cert_manager_certificate, o.ClusterIssuer, o.ClusterIssuer, o.Domain, o.Domain, o.Domain)
	json, err := yaml.YAMLToJSON([]byte(cert))
	assert.NoError(t, err)
	resp, err := o.kubeClient.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", "jx")).Body(json).DoRaw()
	assert.NoError(t, err)
	log.Infof("GOT %s", string(resp))

}
