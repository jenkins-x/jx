// +build integration

package clients_test

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/jx/v2/pkg/vault"

	"github.com/jenkins-x/jx/v2/pkg/kube/serviceaccount"
	rbac_v1 "k8s.io/api/rbac/v1"

	"github.com/Pallinder/go-randomdata"

	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/io/secrets"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
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
		originalJxHome string
		testJxHome     string

		originalKubeCfg string
		testKubeConfig  string

		testNamespace string

		factory    clients.Factory
		kubeClient kubernetes.Interface

		vaultServiceAccountName = "vault-test-sa"

		rootToken      = "snafu"
		vaultPodLabels = map[string]string{
			"app": "vault-test",
		}
		vaultPod        *core_v1.Pod
		rootVaultClient *api.Client
		stopChan        chan struct{}

		err error
	)

	BeforeSuite(func() {
		// See https://www.vaultproject.io/docs/auth/kubernetes for required steps to configure Vault's Kubernetes Auth method

		By("Setting up test logging")
		// comment out to see logging output
		log.SetOutput(ioutil.Discard)
		_ = log.SetLevel("debug")

		By("Setting test specific JX_HOME")
		originalJxHome, testJxHome, err = testhelpers.CreateTestJxHomeDir()
		log.Logger().Debugf("JX_HOME: %s", testJxHome)
		Expect(err).To(BeNil())

		By("Setting test specific KUBECONFIG")
		originalKubeCfg, testKubeConfig, err = testhelpers.CreateTestKubeConfigDir()
		log.Logger().Debugf("KUBECONFIG: %s", testKubeConfig)
		Expect(err).To(BeNil())

		By("Creating client factory")
		factory = clients.NewFactory()
		Expect(factory).NotTo(BeNil())

		By("Creating Kube client")
		kubeClient, _, err = factory.CreateKubeClient()

		By("Creating test namespace")
		testNamespace = strings.ToLower(randomdata.SillyName())
		namespace := core_v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: testNamespace,
			},
		}

		_, err = kubeClient.CoreV1().Namespaces().Create(&namespace)
		Expect(err).To(BeNil())
		log.Logger().Debugf("Test namespace '%s' created", testNamespace)

		By("Creating test service account")
		sa := &core_v1.ServiceAccount{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      vaultServiceAccountName,
				Namespace: testNamespace,
			},
		}
		sa, err = kubeClient.CoreV1().ServiceAccounts(testNamespace).Create(sa)
		Expect(err).To(BeNil())
		log.Logger().Debugf("Test service account '%s' created", sa.Name)

		By("Creating Vault ClusterRoleBinding")
		rb := &rbac_v1.ClusterRoleBinding{
			TypeMeta: meta_v1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: "v1",
			},
			ObjectMeta: meta_v1.ObjectMeta{
				Name: fmt.Sprintf("tokenreview-binding-%s", testNamespace),
			},
			Subjects: []rbac_v1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      vaultServiceAccountName,
					Namespace: testNamespace,
				},
			},
			RoleRef: rbac_v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "system:auth-delegator",
			},
		}
		_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(rb)
		Expect(err).To(BeNil())

		By("Creating test Vault instance")
		vaultPod, err = kubeClient.CoreV1().Pods(testNamespace).Create(buildVaultPod(testNamespace, vaultPodLabels, rootToken))
		Expect(err).To(BeNil())

		By("Waiting for Vault to be ready")
		err = testhelpers.WaitForPod(vaultPod, testNamespace, vaultPodLabels, 10*time.Second, kubeClient)
		Expect(err).To(BeNil())

		By("Port forwarding the Vault port")
		port, err := testhelpers.GetFreePort()
		Expect(err).To(BeNil())

		stopChan, err = testhelpers.PortForward(testNamespace, vaultPod.Name, "8200", strconv.Itoa(port), factory)
		Expect(err).To(BeNil())

		By("Creating a Vault client")
		rootVaultClient, err = buildVaultClient(port, rootToken)
		Expect(err).To(BeNil())

		By("Enabling the KV-2 engine")
		err = rootVaultClient.Sys().Mount("secret", &api.MountInput{
			Type: "kv-v2",
		})

		By("Enabling the Kubernetes auth method")
		authOption := api.EnableAuthOptions{
			Type: "kubernetes",
		}
		err = rootVaultClient.Sys().EnableAuthWithOptions("kubernetes", &authOption)
		Expect(err).Should(BeNil())

		By("Writing Vault test policy")
		policy := `path "secret/*" {capabilities = ["create", "update", "read", "delete"]}`
		err = rootVaultClient.Sys().PutPolicy("jenkins-x-policy", policy)
		Expect(err).Should(BeNil())

		By("Creating Kubernetes Auth role")
		payload := map[string]interface{}{
			"bound_service_account_names":      vaultServiceAccountName,
			"bound_service_account_namespaces": testNamespace,
			"policies":                         "jenkins-x-policy",
			"ttl":                              "24h",
		}
		_, err = rootVaultClient.Logical().Write(fmt.Sprintf("auth/kubernetes/role/%s", vaultServiceAccountName), payload)
		Expect(err).Should(BeNil())

		By("Configuring the Kubernetes auth method")
		token, err := serviceaccount.GetServiceAccountToken(kubeClient, testNamespace, sa.Name)
		Expect(err).Should(BeNil())

		ca, err := serviceaccount.GetServiceAccountCert(kubeClient, testNamespace, sa.Name)
		Expect(err).Should(BeNil())

		config, err := factory.CreateKubeConfig()
		Expect(err).Should(BeNil())

		payload = map[string]interface{}{
			"token_reviewer_jwt": token,
			"kubernetes_host":    config.Host,
			"kubernetes_ca_cert": ca,
		}
		_, err = rootVaultClient.Logical().Write("/auth/kubernetes/config", payload)
		Expect(err).Should(BeNil())
	})

	AfterSuite(func() {
		By("Stopping port forward")
		close(stopChan)

		By("Deleting Vault Pod")
		err = kubeClient.CoreV1().Pods(testNamespace).Delete(vaultPod.Name, &meta_v1.DeleteOptions{})
		Expect(err).To(BeNil())

		err := kubeClient.RbacV1().ClusterRoleBindings().Delete(fmt.Sprintf("tokenreview-binding-%s", testNamespace), &meta_v1.DeleteOptions{})
		Expect(err).To(BeNil())

		err = kubeClient.CoreV1().ServiceAccounts(testNamespace).Delete(vaultServiceAccountName, &meta_v1.DeleteOptions{})
		Expect(err).To(BeNil())

		By("Deleting test namespace")
		err = kubeClient.CoreV1().Namespaces().Delete(testNamespace, &meta_v1.DeleteOptions{})
		Expect(err).To(BeNil())

		By("Resetting JX_HOME")
		err = testhelpers.CleanupTestJxHomeDir(originalJxHome, testJxHome)
		Expect(err).To(BeNil())

		By("Resetting KUBECONFIG")
		err = testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, testKubeConfig)
		Expect(err).To(BeNil())
	})

	Describe("#CreateSystemVaultClient", func() {
		Describe("with no jx-install-config ConfigMap", func() {
			It("CreateSystemVaultClient should fail", func() {
				client, err := factory.CreateSystemVaultClient(testNamespace)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(Equal(fmt.Sprintf("unable to determine Vault type since ConfigMap %s not found in namespace %s", kube.ConfigMapNameJXInstallConfig, testNamespace)))
				Expect(client).Should(BeNil())
			})
		})

		Describe("with jx-install-config ConfigMap, external Vault instance", func() {
			AfterEach(func() {
				err = kubeClient.CoreV1().ConfigMaps(testNamespace).Delete(kube.ConfigMapNameJXInstallConfig, &meta_v1.DeleteOptions{})
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

			It("vault secret store, but not service account name should fail", func() {
				createFunc := func(configMap *core_v1.ConfigMap) error {
					configMap.Data[secrets.SecretsLocationKey] = string(secrets.VaultLocationKind)
					configMap.Data[vault.URL] = rootVaultClient.Address()
					return nil
				}
				_, err = kube.DefaultModifyConfigMap(kubeClient, testNamespace, kube.ConfigMapNameJXInstallConfig, createFunc, nil)
				Expect(err).Should(BeNil())

				client, err := factory.CreateSystemVaultClient(testNamespace)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(ContainSubstring("service account name cannot be empty"))
				Expect(client).Should(BeNil())
			})

			It("vault secret store and service account name should succeed", func() {
				createFunc := func(configMap *core_v1.ConfigMap) error {
					configMap.Data[secrets.SecretsLocationKey] = string(secrets.VaultLocationKind)
					configMap.Data[vault.URL] = rootVaultClient.Address()
					configMap.Data[vault.ServiceAccount] = vaultServiceAccountName
					return nil
				}
				_, err = kube.DefaultModifyConfigMap(kubeClient, testNamespace, kube.ConfigMapNameJXInstallConfig, createFunc, nil)
				Expect(err).Should(BeNil())

				client, err := factory.CreateSystemVaultClient(testNamespace)
				Expect(err).Should(BeNil())
				Expect(client).ShouldNot(BeNil())
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

func buildVaultPod(namespace string, labels map[string]string, token string) *core_v1.Pod {
	return &core_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: "vault-test-",
			Namespace:    namespace,
			Labels:       labels,
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
