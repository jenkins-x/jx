// +build integration

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	testRequest = `
{
  "labels": {
    "created-by-prow": "true",
    "prowJobName": "cdf89f04-98ec-11e9-a846-4ad95a1bb3ab"
  },
  "prowJobSpec": {
    "type": "pullrequest",
    "agent": "tekton",
    "cluster": "default",
    "namespace": "jx",
    "job": "serverless-jenkins",
    "refs": {
      "org": "jenkins-x-quickstarts",
      "repo": "golang-http",
      "repo_link": "https://github.com/jenkins-x-quickstarts/golang-http",
      "base_ref": "master",
      "base_sha": "3f00363d651280ab2a8ee67f395de1689156d762",
      "pulls": [
        {
          "number": 1,
          "sha": "06b5fa6804aa0bd1f4f533010d1b335918a433e2"
        }
      ]
    },
    "report": true,
    "context": "serverless-jenkins",
    "rerun_command": "/test this"
  }
}
`
)

func TestPipelineRunnerIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pipeline Runner Integration Test Suite")
}

var _ = Describe("Pipeline Runner Integration", func() {
	//log.SetOutput(ioutil.Discard)
	var (
		client     *http.Client
		port       int
		err        error
		ctx        context.Context
		cancel     context.CancelFunc
		testOutDir string
	)

	BeforeEach(func() {
		client = &http.Client{}
		Expect(err).Should(BeNil())

		testOutDir, err = ioutil.TempDir("", "jx-pipelinerunner-tests")
		Expect(err).Should(BeNil())

		err = os.Setenv("OUTPUT", testOutDir)
		Expect(err).Should(BeNil())
		err = os.Setenv("NO_APPLY", "true")
		Expect(err).Should(BeNil())

		port, _ = getFreePort()
	})

	AfterEach(func() {
		cancel()
		_ = os.RemoveAll(testOutDir)
		_ = os.Unsetenv("OUTPUT")
		_ = os.Unsetenv("NO_APPLY")
	})

	Describe("when running in meta pipeline mode", func() {
		BeforeEach(func() {
			commonOpts := opts.NewCommonOptionsWithFactory(clients.NewFactory())
			// make sure the Cobra command is created which initialises Viper
			create.NewCmdCreateMetaPipeline(&commonOpts)
			pipelineRunner := PipelineRunnerOptions{
				CommonOptions:        &commonOpts,
				Path:                 "/",
				BindAddress:          "0.0.0.0",
				Port:                 port,
				NoGitCredentialsInit: true,
				UseMetaPipeline:      true,
			}

			go func() {
				var wg sync.WaitGroup
				ctx, cancel = context.WithCancel(context.Background())
				pipelineRunner.startWorkers(ctx, &wg, cancel)
				wg.Wait()
			}()
		})

		It("well formed POST request creates meta pipeline CRDs", func() {
			buffer := bytes.NewBufferString(testRequest)
			request, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/", port), buffer)
			Expect(err).Should(BeNil())
			request.Close = true

			resp, err := client.Do(request)
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

			defer func() {
				_ = resp.Body.Close()
			}()
			data, err := ioutil.ReadAll(resp.Body)

			pipelineRunResponse := PipelineRunResponse{}
			err = json.Unmarshal(data, &pipelineRunResponse)

			var pipelineResource kube.ObjectReference
			for _, resource := range pipelineRunResponse.Resources {
				if resource.Kind == "Pipeline" {
					pipelineResource = resource
				}
			}
			Expect(pipelineResource).ShouldNot(BeNil())
			Expect(pipelineResource.Name).Should(ContainSubstring("meta-jenkins-x-quickstarts"))

			fileInfos, err := ioutil.ReadDir(testOutDir)
			Expect(err).Should(BeNil())
			Expect(len(fileInfos)).Should(Equal(6))

			expectedFiles := []string{"pipeline.yml", "pipeline-run.yml", "structure.yml", "tasks.yml", "resources.yml", "pipelineActivity.yml"}
			for _, file := range fileInfos {
				index := util.StringArrayIndex(expectedFiles, file.Name())
				Expect(index).ShouldNot(Equal(-1))
				expectedFiles = removeElement(expectedFiles, index)
			}
			Expect(len(expectedFiles)).Should(Equal(0))
		})
	})

	Describe("when running in direct pipeline execution mode", func() {
		BeforeEach(func() {
			commonOpts := opts.NewCommonOptionsWithFactory(clients.NewFactory())
			commonOpts.Out = os.Stdout
			// make sure the Cobra command is created which initialises Viper
			create.NewCmdStepCreateTask(&commonOpts)
			pipelineRunner := PipelineRunnerOptions{
				CommonOptions:        &commonOpts,
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

		It("well formed POST request creates Tekton CRDs", func() {
			buffer := bytes.NewBufferString(testRequest)
			request, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/", port), buffer)
			Expect(err).Should(BeNil())
			request.Close = true

			resp, err := client.Do(request)
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

			defer func() {
				_ = resp.Body.Close()
			}()
			data, err := ioutil.ReadAll(resp.Body)

			pipelineRunResponse := PipelineRunResponse{}
			err = json.Unmarshal(data, &pipelineRunResponse)

			var pipelineResource kube.ObjectReference
			for _, resource := range pipelineRunResponse.Resources {
				if resource.Kind == "Pipeline" {
					pipelineResource = resource
				}
			}
			Expect(pipelineResource).ShouldNot(BeNil())
			Expect(pipelineResource.Name).Should(ContainSubstring("jenkins-x-quickstarts-golang"))

			fileInfos, err := ioutil.ReadDir(testOutDir)
			Expect(err).Should(BeNil())
			Expect(len(fileInfos)).Should(Equal(5))

			expectedFiles := []string{"pipeline.yml", "pipeline-run.yml", "structure.yml", "tasks.yml", "resources.yml"}
			for _, file := range fileInfos {
				index := util.StringArrayIndex(expectedFiles, file.Name())
				Expect(index).ShouldNot(Equal(-1))
				expectedFiles = removeElement(expectedFiles, index)
			}
			Expect(len(expectedFiles)).Should(Equal(0))
		})
	})
})

func removeElement(s []string, i int) []string {
	return append(s[:i], s[i+1:]...)
}
