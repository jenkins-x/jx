// +build unit

package boot

import (
	"bytes"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/jenkins-x/jx-logging/pkg/log"
	clients_test "github.com/jenkins-x/jx/v2/pkg/cmd/clients/mocks"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBootVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boot Vault Test Suite")
}

var _ = Describe("#askExternalVaultParameters", func() {
	var testOption StepBootVaultOptions
	var fileHandles util.IOFileHandles
	var testConsole *expect.Console
	var done chan struct{}
	var err error

	BeforeEach(func() {
		// comment out to see logging output
		log.SetOutput(GinkgoWriter)
		_ = log.SetLevel("debug")

		factory := clients_test.NewMockFactory()
		commonOpts := opts.NewCommonOptionsWithFactory(factory)
		testOption = StepBootVaultOptions{
			CommonOptions: &commonOpts,
		}
		Expect(err).Should(BeNil())

		testConsole, err = terminal()
		Expect(err).Should(BeNil())
		fileHandles = util.IOFileHandles{
			Err: testConsole.Tty(),
			In:  testConsole.Tty(),
			Out: testConsole.Tty(),
		}

		done = make(chan struct{})
	})

	AfterEach(func() {
		_ = testConsole.Close()
		<-done
	})

	It("passed requirements are getting updated with required values", func(d Done) {
		requirements := &config.RequirementsConfig{}
		Expect(requirements.Vault.ServiceAccount).To(BeEmpty())
		Expect(requirements.Vault.Namespace).To(BeEmpty())
		Expect(requirements.Vault.URL).To(BeEmpty())
		Expect(requirements.Vault.SecretEngineMountPoint).To(BeEmpty())
		Expect(requirements.Vault.KubernetesAuthPath).To(BeEmpty())

		go func() {
			defer close(done)
			log.Logger().Debug("1")
			_, _ = testConsole.ExpectString("? URL to Vault instance: ")
			_, _ = testConsole.SendLine("https://myvault.com")

			log.Logger().Debug("2")
			_, _ = testConsole.ExpectString("? Authenticating service account: ")
			_, _ = testConsole.SendLine("my-sa")

			log.Logger().Debug("3")
			_, _ = testConsole.ExpectString("? Namespace of authenticating service account: ")
			_, _ = testConsole.SendLine("my-namespace")

			log.Logger().Debug("5")
			_, _ = testConsole.ExpectString("? Mount point for Vault's KV secret engine: ")
			_, _ = testConsole.SendLine("myauth")

			log.Logger().Debug("4")
			_, _ = testConsole.ExpectString("? Path under which to enable Vault's Kubernetes auth plugin: ")
			_, _ = testConsole.SendLine("mysecrets")
		}()

		errorCh := make(chan error)
		go func() {
			err := testOption.askExternalVaultParameters(requirements, fileHandles)
			errorCh <- err
		}()
		Expect(<-errorCh).To(BeNil())

		Expect(requirements.Vault.ServiceAccount).To(Equal("my-sa"))
		Expect(requirements.Vault.Namespace).To(Equal("my-namespace"))
		Expect(requirements.Vault.URL).To(Equal("https://myvault.com"))
		Expect(requirements.Vault.SecretEngineMountPoint).To(Equal("mysecrets"))
		Expect(requirements.Vault.KubernetesAuthPath).To(Equal("myauth"))

		close(d)
	}, 30)
})

func terminal() (*expect.Console, error) {
	buf := new(bytes.Buffer)
	opts := []expect.ConsoleOpt{
		expect.WithStdout(buf),
		expect.WithDefaultTimeout(100 * time.Millisecond),
	}

	console, _, err := vt10x.NewVT10XConsole(opts...)
	if err != nil {
		return nil, err
	}
	return console, nil
}
