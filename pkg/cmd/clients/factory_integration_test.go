package clients_test

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/io/secrets"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestFactory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Factory Suite")
}

var _ = Describe("Vault factory methods", func() {
	var (
		originalJxHome  string
		testJxHome      string
		err             error
		originalKubeCfg string
		testKubeConfig  string
	)

	BeforeSuite(func() {
		// comment out to see logging output
		//log.SetOutput(ioutil.Discard)
		_ = log.SetLevel("debug")

		originalJxHome, testJxHome, err = testhelpers.CreateTestJxHomeDir()
		log.Logger().Debugf("JX_HOME: %s", testJxHome)
		Expect(err).To(BeNil())

		originalKubeCfg, testKubeConfig, err = testhelpers.CreateTestKubeConfigDir()
		log.Logger().Debugf("KUBECONFIG: %s", testKubeConfig)
		Expect(err).To(BeNil())
	})

	AfterSuite(func() {
		err = testhelpers.CleanupTestJxHomeDir(originalJxHome, testJxHome)
		Expect(err).To(BeNil())

		err = testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, testKubeConfig)
		Expect(err).To(BeNil())
	})

	Describe("#CreateSystemVaultClient", func() {
		var (
			factory       clients.Factory
			kubeClient    kubernetes.Interface
			testNamespace string
		)

		BeforeEach(func() {
			factory = clients.NewFactory()
			Expect(factory).NotTo(BeNil())

			kubeClient, _, err = factory.CreateKubeClient()

			testNamespace = strings.ToLower(randomdata.SillyName())
			namespace := core_v1.Namespace{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: testNamespace,
				},
			}

			_, err := kubeClient.CoreV1().Namespaces().Create(&namespace)
			Expect(err).To(BeNil())

			log.Logger().Debugf("test namespace: %s", testNamespace)
		})

		AfterEach(func() {
			err = kubeClient.CoreV1().Namespaces().Delete(testNamespace, &meta_v1.DeleteOptions{})
			Expect(err).To(BeNil())
		})

		Describe("with no jx-install-config ConfigMap", func() {
			It("CreateSystemVaultClient should fail", func() {
				client, err := factory.CreateSystemVaultClient(testNamespace)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(Equal(fmt.Sprintf("unable to determine Vault type since ConfigMap %s not found in namespace %s", kube.ConfigMapNameJXInstallConfig, testNamespace)))
				Expect(client).Should(BeNil())
			})
		})

		Describe("with jx-install-config ConfigMap, external Vault instance", func() {
			var (
				rootToken       = "snafu"
				podName         = "vault-test"
				vaultPod        *core_v1.Pod
				rootVaultClient *api.Client
				stopChan        chan struct{}
			)

			BeforeEach(func() {
				vaultPod, err = kubeClient.CoreV1().Pods(testNamespace).Create(buildPod(testNamespace, podName, rootToken))
				Expect(err).To(BeNil())

				status := vaultPod.Status
				w, err := kubeClient.CoreV1().Pods(testNamespace).Watch(meta_v1.ListOptions{
					Watch:           true,
					ResourceVersion: vaultPod.ResourceVersion,
					LabelSelector:   "app=vault-test",
				})
				Expect(err).To(BeNil())

				func() {
					for {
						select {
						case events, ok := <-w.ResultChan():
							if !ok {
								return
							}
							pod := events.Object.(*core_v1.Pod)
							fmt.Println("Pod status:", pod.Status.Phase)
							status = pod.Status
							if pod.Status.Phase != core_v1.PodPending {
								w.Stop()
							}
						case <-time.After(10 * time.Second):
							fmt.Println("timeout to wait for pod active")
							w.Stop()
						}
					}
				}()
				if status.Phase != core_v1.PodRunning {
					Fail("pod should be running")
				}

				port, err := getFreePort()

				stopChan, err = portForward(testNamespace, vaultPod.Name, strconv.Itoa(port), factory)
				Expect(err).To(BeNil())

				rootVaultClient, err = buildVaultClient(port, rootToken)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				close(stopChan)

				err = kubeClient.CoreV1().ConfigMaps(testNamespace).Delete(kube.ConfigMapNameJXInstallConfig, &meta_v1.DeleteOptions{})
				Expect(err).To(BeNil())

				err = kubeClient.CoreV1().Pods(testNamespace).Delete(vaultPod.Name, &meta_v1.DeleteOptions{})
				Expect(err).To(BeNil())
			})

			It("and local secret store should fail", func() {
				createFunc := func(configMap *core_v1.ConfigMap) error {
					configMap.Data[secrets.SecretsLocationKey] = string(secrets.FileSystemLocationKind)
					return nil
				}
				_, err := kube.DefaultModifyConfigMap(kubeClient, testNamespace, kube.ConfigMapNameJXInstallConfig, createFunc, nil)

				client, err := factory.CreateSystemVaultClient(testNamespace)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(Equal(fmt.Sprintf("unable to create Vault client for secret location kind '%s'", secrets.FileSystemLocationKind)))
				Expect(client).Should(BeNil())
			})

			It("and vault secret store should succeed", func() {
				resp, err := rootVaultClient.Sys().ListPlugins(&api.ListPluginsInput{Type: consts.PluginTypeCredential})
				Expect(err).Should(BeNil())
				log.Logger().Infof("%v", resp)

				authOption := api.EnableAuthOptions{
					Type: "kubernetes",
				}
				err = rootVaultClient.Sys().EnableAuthWithOptions("kubernetes", &authOption)
				Expect(err).Should(BeNil())

				authMethods, _ := rootVaultClient.Sys().ListAuth()
				log.Logger().Infof("%v", authMethods)

				payload := map[string]interface{}{
					"token_reviewer_jwt": "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJqeCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJoYXJkeS1qeC1kZXYtdnQtdG9rZW4tNDRqczQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiaGFyZHktangtZGV2LXZ0Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQudWlkIjoiM2MwMjcxNTgtOGFjNC0xMWVhLWFjNDctNDIwMTBhODAwMTdkIiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Omp4OmhhcmR5LWp4LWRldi12dCJ9.rY8B6e8GQjNeC3UnYBvLDhq9EygOz1_9wjm4fg9gm4ttYupa9h_k-iucjaw_5OBlNl76DHbcPvqnYtnjlEw4dtvEdUlLq3nL-aPT-7g3FInUhz6ebJ4iPBRBvbfPrgAHEnoxNNGsraJvesA0UlGLlbibv6Y2sYwltAVvsmrVusqhPmh9XxRe7U-eyaEq-mmykPhHd1IWWlrx1E4cveFN8whEmB1BUN9eorw9awTdAO6685p_sW2l7Fpox_SlZLFDUhi0uPdiutJQpKnSiSJVxMmpKLdxKIKbJx4dn-KVsCLJtuWUdZKykE2oVJ-6ElWiUP_Csn7kiLUPzFhstMzVIw",
					"kubernetes_host":    "http://foo",
					"kubernetes_ca_cert": "bar",
				}
				_, err = rootVaultClient.Logical().Write("/auth/kubernetes/config", payload)
				Expect(err).Should(BeNil())

				authKube, _ := rootVaultClient.Logical().Read("/auth/kubernetes/config")
				log.Logger().Infof("%v", authKube)

				createFunc := func(configMap *core_v1.ConfigMap) error {
					configMap.Data[secrets.SecretsLocationKey] = string(secrets.VaultLocationKind)
					configMap.Data[kube.VaultURL] = rootVaultClient.Address()
					return nil
				}
				_, err = kube.DefaultModifyConfigMap(kubeClient, testNamespace, kube.ConfigMapNameJXInstallConfig, createFunc, nil)
				Expect(err).Should(BeNil())

				//client, err := factory.CreateSystemVaultClient(testNamespace)
				//Expect(err).Should(BeNil())
				//Expect(client).ShouldNot(BeNil())
			})
		})
	})
})

func buildVaultClient(port int, token string) (*api.Client, error) {
	//Create a client that talks to the server, initially authenticating with the root token
	conf := api.DefaultConfig()
	conf.Address = fmt.Sprintf("http://localhost:%d", port)

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create vault client")
	}
	client.SetToken(token)

	return client, err
}

func buildPod(namespace string, name string, token string) *core_v1.Pod {
	return &core_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: name,
			Namespace:    namespace,
			Labels: map[string]string{
				"app": "vault-test",
			},
		},
		Spec: core_v1.PodSpec{
			RestartPolicy: core_v1.RestartPolicyNever,
			Containers: []core_v1.Container{
				{
					Name:            "vault-test-instance",
					Image:           "vault:1.4.1",
					ImagePullPolicy: core_v1.PullIfNotPresent,
					Ports: []core_v1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8200,
						},
					},
					Env: []core_v1.EnvVar{
						{
							Name:  "VAULT_DEV_ROOT_TOKEN_ID",
							Value: token,
						},
					},
				},
			},
		},
	}
}

func portForward(namespace string, podName string, forwardPort string, factory clients.Factory) (chan struct{}, error) {
	config, err := factory.CreateKubeConfig()
	if err != nil {
		return nil, err
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(config.Host, "https:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%s:8200", forwardPort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
		}
		if len(errOut.String()) != 0 {
			log.Logger().Error(errOut.String())
		} else if len(out.String()) != 0 {
			log.Logger().Info(out.String())
		}
	}()

	go func() {
		_ = forwarder.ForwardPorts()
	}()

	return stopChan, nil
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = l.Close()
	}()
	return l.Addr().(*net.TCPAddr).Port, nil
}
