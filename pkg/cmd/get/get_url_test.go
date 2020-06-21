// +build unit

package get_test

import (
	"os"
	"testing"

	envV1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/get"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNameSpace   = "jx-url-test"
	exposeURL       = "https://jx-expose.com"
	ingressIP       = "10.10.10.10"
	ingressHostName = "https://jx-ingress-host.com"
	ingressTLSHost  = "jx-ingress-tls.com"
	ingressRuleHost = "jx-ingress-rule.com"
)

// These testcases are not needed here, they can be moved to the services package tests
var getURLTests = []struct {
	desc            string
	namespace       string
	svcAnnotations  map[string]string
	Name            string
	svcPorts        []v1.ServicePort
	ingressIP       string
	ingressHostName string
	ingressTLSHost  string
	ingressRuleHost string
}{
	{"Case1: Error when kubeclient cannot be created", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	{"Case 2: Error when namespace cannot be found", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	{"Case 3: Namespace cannot be found for an environment", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	{"Case 4: Error with finding service urls", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	{"Case 5: No flag errors", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	{"Case 5: No flag errors", testNameSpace,
		map[string]string{
			"fabric8.io/exposeUrl": exposeURL,
		}, "test-service", nil, "", "", "", ""},
	//{"Service: ServiceType Loadbalancer and ingressIp is present (Port 443)", testNameSpace,
	//	nil, "test-service", []v1.ServicePort{{
	//		Port: 443,
	//	}}, ingressIP, "", "", ""},
	//{"Service:  ServiceType Loadbalancer and Only ingress hostname is present", testNameSpace,
	//	nil, "test-service", nil, "", ingressHostName,
	//	"", ""},
	//{"Service:  ServiceType Loadbalancer and Both ingress hostname and IP are present", testNameSpace,
	//	nil, "test-service", nil, ingressIP, ingressHostName,
	//	"", ""},
	//{"No Service, trying ingress: Non-empty TLS hosts", testNameSpace,
	//	nil, "test-service", nil, "", "",
	//	ingressTLSHost, ingressRuleHost},
	//{"No Service, trying ingress: Empty TLS hosts, checking rules", testNameSpace,
	//	nil, "test-service", nil, "", "",
	//	"", ingressRuleHost},
	//{"No Service, No ingress, so no url", testNameSpace,
	//	nil, "test-service", nil, "", "",
	//	"", ""},
}

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

func environment() *envV1.Environment {
	return &envV1.Environment{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       envV1.EnvironmentSpec{},
		Status:     envV1.EnvironmentStatus{},
	}
}

// Basic unit tests to check that all the different flags properly for jx version sub command
func Test_ExecuteGetUrls(t *testing.T) {
	for _, v := range getURLTests {
		t.Run(v.desc, func(t *testing.T) {
			// fakeout the output for the tests
			out := &testhelpers.FakeOut{}
			commonOpts := opts.NewCommonOptionsWithTerm(clients.NewFactory(), os.Stdin, out, os.Stderr)

			// Set batchmode to true for tests
			commonOpts.BatchMode = true

			// Set dev namespace
			commonOpts.SetDevNamespace(v.namespace)

			// Fake kubernetes client
			client := fake.NewSimpleClientset(service(v.Name, v.namespace, v.ingressIP, v.ingressHostName,
				v.svcAnnotations, v.svcPorts), ingress(v.Name, v.namespace, v.ingressTLSHost, v.ingressRuleHost))
			commonOpts.SetKubeClient(client)

			// Fake jx client
			jxClient := v1fake.NewSimpleClientset(environment())
			commonOpts.SetJxClient(jxClient)

			command := get.NewCmdGetURL(commonOpts)
			// Different test cases for args passed to the command is what we should test here
			//command.SetArgs()
			err := command.Execute()

			// Execution should not error out
			assert.NoError(t, err, "execute get urls")

			// Check if annotations are present
			if len(v.svcAnnotations) > 0 {
				assert.Contains(t, out.GetOutput(), exposeURL)
				assert.NotContains(t, out.GetOutput(), ingressIP)
				assert.NotContains(t, out.GetOutput(), ingressHostName)
				assert.NotContains(t, out.GetOutput(), ingressTLSHost)
				assert.NotContains(t, out.GetOutput(), ingressRuleHost)
			} else {
				if v.ingressHostName != "" || v.ingressIP != "" {
					if v.ingressIP != "" {
						assert.NotContains(t, out.GetOutput(), exposeURL)
						assert.Contains(t, out.GetOutput(), ingressIP)
						if v.ingressHostName != "" {
							// Even if hostname is present, it should not show, only ingress IP is shown
							assert.NotContains(t, out.GetOutput(), ingressHostName)
						}
						assert.NotContains(t, out.GetOutput(), ingressTLSHost)
						assert.NotContains(t, out.GetOutput(), ingressRuleHost)
					}

					if v.ingressHostName != "" && v.ingressIP == "" {
						assert.NotContains(t, out.GetOutput(), exposeURL)
						assert.NotContains(t, out.GetOutput(), ingressIP)
						assert.Contains(t, out.GetOutput(), ingressHostName)
						assert.NotContains(t, out.GetOutput(), ingressTLSHost)
						assert.NotContains(t, out.GetOutput(), ingressRuleHost)
					}

				} else {
					if v.ingressTLSHost != "" {
						assert.NotContains(t, out.GetOutput(), exposeURL)
						assert.NotContains(t, out.GetOutput(), ingressIP)
						assert.NotContains(t, out.GetOutput(), ingressHostName)
						assert.Contains(t, out.GetOutput(), ingressTLSHost)
						if v.ingressRuleHost != "" {
							// Even if this is present, it wont be shown
							assert.NotContains(t, out.GetOutput(), ingressRuleHost)
						}
					}

					if v.ingressRuleHost != "" && v.ingressTLSHost == "" {
						assert.NotContains(t, out.GetOutput(), exposeURL)
						assert.NotContains(t, out.GetOutput(), ingressIP)
						assert.NotContains(t, out.GetOutput(), ingressHostName)
						assert.NotContains(t, out.GetOutput(), ingressTLSHost)
						assert.Contains(t, out.GetOutput(), ingressRuleHost)
					}
					if v.ingressRuleHost == "" && v.ingressTLSHost == "" {
						assert.NotContains(t, out.GetOutput(), exposeURL)
						assert.NotContains(t, out.GetOutput(), ingressIP)
						assert.NotContains(t, out.GetOutput(), ingressHostName)
						assert.NotContains(t, out.GetOutput(), ingressTLSHost)
						assert.NotContains(t, out.GetOutput(), ingressRuleHost)
					}
				}
			}
		})
	}
}
