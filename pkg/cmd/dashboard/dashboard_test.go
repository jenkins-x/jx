package dashboard_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/dashboard"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	nv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type FakeBrowser struct{}

func (*FakeBrowser) Open() error {
	return nil
}

const testNamespace = "jx"

func TestNewCmdDashboard(t *testing.T) {
	testCases := []struct {
		description string
		hasError    bool
		NoBrowser   bool
		secret      map[string][]byte
		host        string
	}{
		{
			description: "Test Case 1 - do not open browser",
			hasError:    false,
			NoBrowser:   true,
			host:        "hook-jx.1.2.3.4.nip.io",
		},
		{
			description: "Test Case 2 - open browser",
			hasError:    false,
			NoBrowser:   false,
			secret: map[string][]byte{
				"username": []byte("username"),
				"password": []byte("password"),
			},
			host: "hook-jx.1.2.3.4.nip.io",
		},
		{
			description: "Test Case 3 - nil secret",
			hasError:    false,
			NoBrowser:   false,
			secret:      nil,
			host:        "hook-jx.1.2.3.4.nip.io",
		},
		{
			description: "Test Case 4 - empty username in secret",
			hasError:    false,
			NoBrowser:   false,
			secret: map[string][]byte{
				"username": []byte(""),
				"password": []byte("password"),
			},
			host: "hook-jx.1.2.3.4.nip.io",
		},
		{
			description: "Test Case 5 - empty password in secret",
			hasError:    false,
			NoBrowser:   false,
			secret: map[string][]byte{
				"username": []byte("username"),
				"password": []byte(""),
			},
			host: "hook-jx.1.2.3.4.nip.io",
		},
		{
			description: "Test Case 6 - invalid url",
			hasError:    true,
			NoBrowser:   false,
			secret: map[string][]byte{
				"username": []byte("username"),
				"password": []byte("password"),
			},
			host: "invalid url",
		},
	}

	for _, tt := range testCases {
		t.Logf("Running Test case %s", tt.description)
		kubeClient := fake.NewSimpleClientset(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			},
			&nv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jx-pipelines-visualizer",
					Namespace: testNamespace,
				},
				Spec: nv1.IngressSpec{
					Rules: []nv1.IngressRule{
						{
							Host: tt.host,
							IngressRuleValue: nv1.IngressRuleValue{
								HTTP: &nv1.HTTPIngressRuleValue{
									Paths: []nv1.HTTPIngressPath{
										{
											Path: "",
											Backend: nv1.IngressBackend{
												Service: &nv1.IngressServiceBackend{
													Name: "hook",
													Port: nv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			&v1.Secret{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jx-basic-auth-user-password",
					Namespace: testNamespace,
				},
				Data: tt.secret,
			})
		os.Setenv("KUBECONFIG", "testdata/kubeconfig")

		_, o := dashboard.NewCmdDashboard()
		o.KubeClient = kubeClient
		o.Namespace = testNamespace
		o.NoBrowser = tt.NoBrowser
		if !o.NoBrowser {
			o.BrowserHandler = &FakeBrowser{}
		}
		err := o.Run()
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
