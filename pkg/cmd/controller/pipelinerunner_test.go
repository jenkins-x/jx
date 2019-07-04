// +build !integration

package controller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/log"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPipelineRunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pipeline Runner Test Suite")
}

var _ = Describe("Pipeline Runner", func() {
	Describe("when running", func() {
		var (
			client *http.Client
			port   int
			err    error
			ctx    context.Context
			cancel context.CancelFunc
		)

		BeforeEach(func() {
			log.SetOutput(ioutil.Discard)
			client = &http.Client{}
			Expect(err).Should(BeNil())

			port, _ = getFreePort()
			pipelineRunner := PipelineRunnerOptions{
				CommonOptions:        &opts.CommonOptions{},
				Path:                 "/",
				BindAddress:          "0.0.0.0",
				Port:                 port,
				NoGitCredentialsInit: true,
			}

			go func() {
				var wg sync.WaitGroup
				ctx, cancel = context.WithCancel(context.Background())
				pipelineRunner.startWorkers(ctx, &wg, cancel)
				wg.Wait()
			}()
		})

		AfterEach(func() {
			cancel()
		})

		It("GET requests return with HTTP 200 and request POST", func() {
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", port))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(200))

			defer resp.Body.Close()
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(string(htmlData)).Should(ContainSubstring("please POST JSON "))
		})

		It("/health returns HTTP 204", func() {
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d%s", port, healthPath))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusNoContent))
		})

		It("/ready returns HTTP 204", func() {
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d%s", port, readyPath))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusNoContent))
		})

		It("POST returns HTTP 400", func() {
			var json = []byte("{\"foo\":\"bar\"}")
			resp, err := client.Post(fmt.Sprintf("http://localhost:%d/", port), "application/json", bytes.NewBuffer(json))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))

			defer resp.Body.Close()
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(string(htmlData)).Should(ContainSubstring("could not start pipeline"))
		})
	})
})
