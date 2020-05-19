package clients_test

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/consts"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/io/secrets"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/ory/dockertest/v3"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestFactory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git CLI Test Suite")
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
				dockerPool      *dockertest.Pool
				vaultResource   *dockertest.Resource
				rootVaultClient *api.Client
			)

			BeforeEach(func() {
				dockerPool, vaultResource, rootVaultClient, err = createTestVaultInstance()
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				err = kubeClient.CoreV1().ConfigMaps(testNamespace).Delete(kube.ConfigMapNameJXInstallConfig, &meta_v1.DeleteOptions{})
				Expect(err).To(BeNil())

				err = dockerPool.Purge(vaultResource)
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
				resp, _ := rootVaultClient.Sys().ListPlugins(&api.ListPluginsInput{Type: consts.PluginTypeCredential})
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
				_, err := kube.DefaultModifyConfigMap(kubeClient, testNamespace, kube.ConfigMapNameJXInstallConfig, createFunc, nil)
				Expect(err).Should(BeNil())

				//client, err := factory.CreateSystemVaultClient(testNamespace)
				//Expect(err).Should(BeNil())
				//Expect(client).ShouldNot(BeNil())
			})
		})
	})
})

func createTestVaultInstance() (*dockertest.Pool, *dockertest.Resource, *api.Client, error) {
	rootToken := "snafu"

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "unable to create docker pool")
	}

	options := &dockertest.RunOptions{
		Repository:   "vault",
		Tag:          "1.4.1",
		ExposedPorts: []string{"8200"},
		Env:          []string{fmt.Sprintf("VAULT_DEV_ROOT_TOKEN_ID=%s", rootToken)},
	}

	resource, err := pool.RunWithOptions(options)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "unable to start vault container")
	}

	vaultURL := "http://localhost:8200"
	addr := resource.GetHostPort("8200/tcp")
	if addr != "" {
		vaultURL = fmt.Sprintf("http://%s", addr)
	}

	//Create a client that talks to the server, initially authenticating with the root token
	conf := api.DefaultConfig()
	conf.Address = vaultURL

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "unable to create vault client")
	}
	client.SetToken(rootToken)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		_, err := client.Sys().SealStatus()
		return err
	}); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "timeout waiting for vault")
	}

	return pool, resource, client, err
}
