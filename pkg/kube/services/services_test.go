// +build unit

package services_test

import (
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/kube/services"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNameSpace   = "jx-service-test"
	testServiceName = "service-test"
	exposeURL       = "https://jx-expose.com"
	ingressIP       = "10.10.10.10"
	ingressHostName = "https://jx-ingress-host.com"
	ingressTLSHost  = "jx-ingress-tls.com"
	ingressRuleHost = "jx-ingress-rule.com"
)

func service(name, ns, ingressIP, ingressHostName string, annotations map[string]string,
	ports []v1.ServicePort) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Ports: ports,
			Type:  v1.ServiceTypeLoadBalancer,
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{
						IP:       ingressIP,
						Hostname: ingressHostName,
					},
				},
			},
		},
	}
}

func ingress(name, ns, ingressTLSHost, ingressRuleHost string) *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts: []string{ingressTLSHost},
			}},
			Rules: []v1beta1.IngressRule{{
				Host: ingressRuleHost,
			}},
		},
	}
}

func namespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNameSpace,
		},
	}
}

var findServiceURLTests = []struct {
	desc            string
	namespace       string
	svcAnnotations  map[string]string
	name            string
	svcPorts        []v1.ServicePort
	ingressIP       string
	ingressHostName string
	ingressTLSHost  string
	ingressRuleHost string
	urlPresent      bool
}{
	{"Service: service annotation is present", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", "", true},
	{"Service: ServiceType Loadbalancer and ingressIp is present (Port 443)", testNameSpace,
		nil, "test-service", []v1.ServicePort{{
			Port: 443,
		}}, ingressIP, "", "", "", true},
	{"Service:  ServiceType Loadbalancer and Only ingress hostname is present", testNameSpace,
		nil, "test-service", nil, "", ingressHostName,
		"", "", true},
	{"Service:  ServiceType Loadbalancer and Both ingress hostname and IP are present", testNameSpace,
		nil, "test-service", nil, ingressIP, ingressHostName,
		"", "", true},
	{"No Service, trying ingress: Non-empty TLS hosts", testNameSpace,
		nil, "test-service", nil, "", "",
		ingressTLSHost, ingressRuleHost, true},
	{"No Service, trying ingress: Empty TLS hosts, checking rules", testNameSpace,
		nil, "test-service", nil, "", "",
		"", ingressRuleHost, true},
	{"No Service, No ingress, so no url", testNameSpace,
		nil, "test-service", nil, "", "",
		"", "", false},
}

func setUpKubeClient() kubernetes.Interface {
	// Fake kubernetes client
	client := fake.NewSimpleClientset(service(testServiceName, testNameSpace, "10.10.10.10",
		"", nil, nil))
	return client
}

var findServicesTests = []struct {
	desc            string
	namespace       string
	svcAnnotations  map[string]string
	name            string
	svcPorts        []v1.ServicePort
	ingressIP       string
	ingressHostName string
	ingressTLSHost  string
	ingressRuleHost string
	servicePresent  bool
}{
	{"Service: service annotation is present", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", "", true},
	{"Service: ServiceType Loadbalancer and ingressIp is present (Port 443)", testNameSpace,
		nil, "test-service", []v1.ServicePort{{
			Port: 443,
		}}, ingressIP, "", "", "", true},
	{"Service:  ServiceType Loadbalancer and Only ingress hostname is present", testNameSpace,
		nil, "test-service", nil, "", ingressHostName,
		"", "", true},
	{"Service:  ServiceType Loadbalancer and Both ingress hostname and IP are present", testNameSpace,
		nil, "test-service", nil, ingressIP, ingressHostName,
		"", "", true},
	{"No Service, trying ingress: Non-empty TLS hosts", testNameSpace,
		nil, "test-service", nil, "", "",
		ingressTLSHost, ingressRuleHost, true},
	{"No Service, trying ingress: Empty TLS hosts, checking rules", testNameSpace,
		nil, "test-service", nil, "", "",
		"", ingressRuleHost, true},
	{"No Service, No ingress", testNameSpace,
		nil, "test-service", nil, "", "",
		"", "", false},
}

func Test_GetServices(t *testing.T) {
	for _, v := range findServicesTests {
		t.Run(v.desc, func(t *testing.T) {
			// Fake kubernetes client
			client := fake.NewSimpleClientset()
			if v.servicePresent {
				client = fake.NewSimpleClientset(service(v.name, v.namespace, v.ingressIP, v.ingressHostName,
					v.svcAnnotations, v.svcPorts), ingress(v.name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
			}
			svcs, err := services.GetServices(client, v.namespace)
			assert.NoError(t, err, "getting service url")
			if v.servicePresent {
				assert.NotEmpty(t, svcs)
			} else {
				assert.Empty(t, svcs)
			}
		})
	}
}

func Test_GetServicesByName(t *testing.T) {
	client := setUpKubeClient()
	svcs, err := services.GetServicesByName(client, testNameSpace, []string{testServiceName})
	assert.NoError(t, err, "getting services")
	assert.NotEmpty(t, svcs)
}

func Test_GetServiceNames(t *testing.T) {
	client := setUpKubeClient()
	svcs, err := services.GetServiceNames(client, testNameSpace, "service")
	assert.NoError(t, err, "getting services")
	assert.NotEmpty(t, svcs)
}

func Test_GetServiceURL(t *testing.T) {
	url := services.GetServiceURL(service(testServiceName, testNameSpace, "",
		"", nil, nil))
	assert.Empty(t, url)
}

func Test_GetServiceURLFromName(t *testing.T) {
	client := setUpKubeClient()
	svcs, err := services.GetServiceURLFromName(client, testServiceName, testNameSpace)
	assert.NoError(t, err, "getting services")
	assert.NotEmpty(t, svcs)
}

func Test_FindServiceSchemePort(t *testing.T) {
	client := setUpKubeClient()
	scheme, port, err := services.FindServiceSchemePort(client, testNameSpace, testServiceName)
	assert.NoError(t, err, "getting services")
	assert.Empty(t, scheme)
	assert.Empty(t, port)
}

func Test_FindService(t *testing.T) {
	for _, v := range findServiceURLTests {
		t.Run(v.desc, func(t *testing.T) {
			// Fake kubernetes client
			client := fake.NewSimpleClientset(service(v.name, v.namespace, v.ingressIP, v.ingressHostName,
				v.svcAnnotations, v.svcPorts), ingress(v.name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
			// Create a dummy namespace
			_, err := client.CoreV1().Namespaces().Create(namespace())
			assert.NoError(t, err, "creating namespace for test")
			urls, err := services.FindService(client, v.namespace)
			assert.NoError(t, err, "getting service url")
			if v.urlPresent {
				assert.NotEmpty(t, urls)
			} else {
				assert.Empty(t, urls)
			}

		})
	}

	client := setUpKubeClient()
	// Create a dummy namespace
	_, err := client.CoreV1().Namespaces().Create(namespace())
	assert.NoError(t, err, "creating namespace for test")
	svcs, err := services.FindService(client, testServiceName)
	assert.NoError(t, err, "finding service")
	assert.NotEmpty(t, svcs)
}

func Test_FindServiceURL(t *testing.T) {
	for _, v := range findServiceURLTests {
		t.Run(v.desc, func(t *testing.T) {
			// Fake kubernetes client
			client := fake.NewSimpleClientset(service(v.name, v.namespace, v.ingressIP, v.ingressHostName,
				v.svcAnnotations, v.svcPorts), ingress(v.name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
			urls, err := services.FindServiceURL(client, v.namespace, v.name)
			assert.NoError(t, err, "getting service url")
			if v.urlPresent {
				assert.NotEmpty(t, urls)
			} else {
				assert.Empty(t, urls)
			}

		})
	}
}

func Test_FindServiceURLs(t *testing.T) {
	for _, v := range findServiceURLTests {
		t.Run(v.desc, func(t *testing.T) {
			// Fake kubernetes client
			client := fake.NewSimpleClientset(service(v.name, v.namespace, v.ingressIP, v.ingressHostName,
				v.svcAnnotations, v.svcPorts), ingress(v.name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
			urls, err := services.FindServiceURLs(client, v.namespace)
			assert.NoError(t, err, "getting service url")
			if v.urlPresent {
				assert.NotEmpty(t, urls)
			} else {
				assert.Empty(t, urls)
			}
		})
	}
}

func Test_AnnotateServicesWithBasicAuth(t *testing.T) {
	client := setUpKubeClient()
	err := services.AnnotateServicesWithBasicAuth(client, testNameSpace, "")
	assert.NoError(t, err, "getting services")

}

func Test_AnnotateServicesWithCertManagerIssuer(t *testing.T) {
	client := setUpKubeClient()
	svcs, err := services.AnnotateServicesWithCertManagerIssuer(client, testNameSpace, "", false, testServiceName)
	assert.NoError(t, err, "getting services")
	assert.Empty(t, svcs)
}

func Test_CleanServiceAnnotations(t *testing.T) {
	client := setUpKubeClient()
	err := services.CleanServiceAnnotations(client, testNameSpace, testServiceName)
	assert.NoError(t, err, "cleaning service annotation")
}

func Test_CreateServiceLink(t *testing.T) {
	client := setUpKubeClient()
	err := services.CreateServiceLink(client, testNameSpace, "jx-target", "service-2", "")
	assert.NoError(t, err, "creating service link")
}

func Test_IsServicePresent(t *testing.T) {
	client := setUpKubeClient()
	svcs, err := services.IsServicePresent(client, testServiceName, testNameSpace)
	assert.NoError(t, err, "getting services")
	assert.NotEmpty(t, svcs)
}

func Test_ServiceAppName(t *testing.T) {
	svcName := services.ServiceAppName(service(testServiceName, testNameSpace, "",
		"", nil, nil))
	assert.NotEmpty(t, svcName)
}

var portCases = []struct {
	desc         string
	servicePorts []v1.ServicePort
	scheme       string
	port         string
}{
	{"Default (HTTP)", []v1.ServicePort{
		{
			Name:     "http",
			Protocol: "TCP",
			Port:     80,
		},
	}, "http", "80"},
	{"HTTPs", []v1.ServicePort{
		{
			Name:     "https",
			Protocol: "TCP",
			Port:     443,
		},
	}, "https", "443"},
	{"HTTPsFirst", []v1.ServicePort{
		{
			Name:     "http",
			Protocol: "TCP",
			Port:     80,
		},
		{
			Name:     "https",
			Protocol: "TCP",
			Port:     443,
		},
	}, "https", "443"},
	{"HTTPsOdd", []v1.ServicePort{
		{
			Name:     "brian",
			Protocol: "TCP",
			Port:     443,
		},
	}, "https", "443"},
	{"HTTPsNamed", []v1.ServicePort{
		{
			Name:     "dave",
			Protocol: "UDP",
			Port:     800,
		},
		{
			Name:     "brian",
			Protocol: "TCP",
			Port:     444,
		},
		{
			Name:     "https",
			Protocol: "TCP",
			Port:     443,
		},
	}, "https", "443"},
	{"HTTPNamed", []v1.ServicePort{
		{
			Name:     "dave",
			Protocol: "UDP",
			Port:     800,
		},
		{
			Name:     "brian",
			Protocol: "TCP",
			Port:     444,
		},
		{
			Name:     "http",
			Protocol: "TCP",
			Port:     8083,
		},
	}, "http", "8083"},
	{"HTTPNotNamed", []v1.ServicePort{
		{
			Name:     "http",
			Protocol: "TCP",
			Port:     8088,
		},
		{
			Name:     "alan",
			Protocol: "TCP",
			Port:     80,
		},
	}, "http", "80"},
	{"NamedPrefHttps", []v1.ServicePort{
		{
			Name:     "ssh",
			Protocol: "UDP",
			Port:     22,
		},
		{
			Name:     "hiddenhttp",
			Protocol: "TCP",
			Port:     8083,
		},
		{
			Name:     "sctp-tunneling",
			Protocol: "TCP",
			Port:     9899,
		},
		{
			Name:     "https",
			Protocol: "TCP",
			Port:     8443,
		},
	}, "https", "8443"},
	{"Inconclusive", []v1.ServicePort{
		{
			Name:     "ssh",
			Protocol: "UDP",
			Port:     22,
		},
		{
			Name:     "hiddenhttp",
			Protocol: "TCP",
			Port:     8083,
		},
		{
			Name:     "sctp-tunneling",
			Protocol: "TCP",
			Port:     9899,
		},
	}, "", ""},
}

func Test_ExtractServiceSchemePort(t *testing.T) {
	for _, v := range portCases {
		t.Run(v.desc, func(t *testing.T) {
			scheme, port, _ := services.ExtractServiceSchemePort(service("", "", "", "",
				nil, v.servicePorts))
			assert.Equal(t, v.scheme, scheme)
			assert.Equal(t, v.port, port)
		})
	}
}

func Test_IngressHost(t *testing.T) {
	host := services.IngressHost(ingress("ingress-1", testNameSpace, "", ""))
	assert.Empty(t, host)
}

func Test_IngressURL(t *testing.T) {
	url := services.IngressURL(ingress("ingress-1", testNameSpace, "", ""))
	assert.Empty(t, url)
}

func Test_FindIngressURL(t *testing.T) {
	//for _, v := range findServiceURLTests {
	//	t.Run(v.desc, func(t *testing.T) {
	//		// Fake kubernetes client
	//		client := fake.NewSimpleClientset(service(v.name, v.namespace, v.ingressIP, v.ingressHostName,
	//			v.svcAnnotations, v.svcPorts), ingress(v.name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
	//		urls, err := services.FindIngressURL(client, v.namespace, v.name)
	//		assert.NoError(t, err, "getting service url")
	//		if v.urlPresent {
	//			assert.NotEmpty(t, urls)
	//		} else {
	//			assert.Empty(t, urls)
	//		}
	//	})
	//}

	client := setUpKubeClient()
	svcs, err := services.FindIngressURL(client, testNameSpace, "ingress-1")
	assert.NoError(t, err, "finding ingress url")
	assert.Empty(t, svcs)
}
