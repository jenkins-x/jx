// +build integration

package serviceaccount_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/jx/v2/pkg/kube/serviceaccount"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/kubernetes/staging/src/k8s.io/api/core/v1"

	"github.com/Pallinder/go-randomdata"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestServiceAccounts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceAccounts Suite")
}

var _ = Describe("ServiceAccounts methods", func() {
	var (
		originalJxHome string
		testJxHome     string

		originalKubeCfg string
		testKubeConfig  string

		testNamespace string

		factory    clients.Factory
		kubeClient kubernetes.Interface

		testServiceAccountName = "service-account-integration-test-sa"

		err error
	)

	BeforeSuite(func() {
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
				Name:      testServiceAccountName,
				Namespace: testNamespace,
			},
		}
		sa, err = kubeClient.CoreV1().ServiceAccounts(testNamespace).Create(sa)
		Expect(err).To(BeNil())
		log.Logger().Debugf("Test service account '%s' created", sa.Name)

		err = util.Retry(60*time.Second, func() error {
			secretList, err := kubeClient.CoreV1().Secrets(testNamespace).List(meta_v1.ListOptions{})
			if err != nil {
				return err
			}
			for _, secret := range secretList.Items {
				annotations := secret.ObjectMeta.Annotations
				for k, v := range annotations {
					if k == v1.ServiceAccountNameKey && v == testServiceAccountName {
						return nil
					}
				}
			}
			return errors.New("unable to find secret")
		})
		Expect(err).To(BeNil())
	})

	AfterSuite(func() {
		By("Deleting test service account")
		err = kubeClient.CoreV1().ServiceAccounts(testNamespace).Delete(testServiceAccountName, &meta_v1.DeleteOptions{})
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

	Describe("#GetServiceAccountToken", func() {
		It("succeeds with valid service account", func() {
			jwt, err := serviceaccount.GetServiceAccountToken(kubeClient, testNamespace, testServiceAccountName)
			Expect(err).To(BeNil())
			Expect(jwt).NotTo(BeEmpty())
		})

		It("fails with unknown service account", func() {
			ca, err := serviceaccount.GetServiceAccountToken(kubeClient, testNamespace, "fubar")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(fmt.Sprintf("no token found for service account fubar in namespace %s", testNamespace)))
			Expect(ca).To(BeEmpty())
		})
	})

	Describe("#GetServiceAccountCert", func() {
		It("succeeds with valid service account", func() {
			ca, err := serviceaccount.GetServiceAccountCert(kubeClient, testNamespace, testServiceAccountName)
			Expect(err).To(BeNil())
			Expect(ca).NotTo(BeEmpty())
			lines := strings.Split(ca, "\n")
			Expect(lines[0]).To(HavePrefix("-----BEGIN CERTIFICATE-----"))
			Expect(lines[len(lines)-2]).To(HaveSuffix("-----END CERTIFICATE-----"))
			Expect(lines[len(lines)-1]).To(BeEmpty())
		})

		It("fails with unknown service account", func() {
			ca, err := serviceaccount.GetServiceAccountCert(kubeClient, testNamespace, "fubar")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(fmt.Sprintf("no ca.crt found for service account fubar in namespace %s", testNamespace)))
			Expect(ca).To(BeEmpty())
		})
	})
})
