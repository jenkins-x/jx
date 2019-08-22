package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testRequestWithoutProwJobName = `
{
  "labels": {
    "created-by-prow": "true"
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
	testRequestMissingPullRefs = `
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
    "report": true,
    "context": "serverless-jenkins",
    "rerun_command": "/test this"
  }
}
`
)

func TestPipelineRunner(t *testing.T) {
	// See https://github.com/jenkins-x/jx/issues/5167
	t.SkipNow()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pipeline Runner Test Suite")
}

var _ = Describe("Pipeline Runner", func() {
	BeforeSuite(func() {
		log.SetOutput(ioutil.Discard)
	})

	Describe("when running", func() {
		var (
			client *http.Client
			port   int
			err    error
			ctx    context.Context
			cancel context.CancelFunc
		)

		BeforeEach(func() {
			client = &http.Client{}
			Expect(err).Should(BeNil())

			jxClient := fake.NewSimpleClientset()
			Expect(err).Should(BeNil())

			port, _ = getFreePort()
			controller := controller{
				path:            "/",
				bindAddress:     "127.0.0.1",
				port:            port,
				useMetaPipeline: true,
				jxClient:        jxClient,
				ns:              "jx",
			}

			go func() {
				var wg sync.WaitGroup
				ctx, cancel = context.WithCancel(context.Background())
				controller.startWorkers(ctx, &wg, cancel)
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

		It("POST returns HTTP 400 for invalid JSON", func() {
			var json = []byte("{\"foo\":\"bar\"}")
			resp, err := client.Post(fmt.Sprintf("http://localhost:%d/", port), "application/json", bytes.NewBuffer(json))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))

			defer resp.Body.Close()
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(string(htmlData)).Should(ContainSubstring("could not start pipeline"))
		})

		It("POST returns HTTP 400 for missing pull refs", func() {
			resp, err := client.Post(fmt.Sprintf("http://localhost:%d/", port), "application/json", bytes.NewBuffer([]byte(testRequestMissingPullRefs)))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))

			defer resp.Body.Close()
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(string(htmlData)).Should(ContainSubstring("no prowJobSpec.refs passed"))
		})

		It("POST returns HTTP 400 for missing prow job name", func() {
			resp, err := client.Post(fmt.Sprintf("http://localhost:%d/", port), "application/json", bytes.NewBuffer([]byte(testRequestWithoutProwJobName)))
			Expect(err).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))

			defer resp.Body.Close()
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(string(htmlData)).Should(ContainSubstring("unable to find prow job name in pipeline request"))
		})
	})

	Describe("#getSourceURL", func() {
		var (
			port           int
			testController controller
			expectedURL    = "http://github.com/jenkins-x/jx.git"
		)

		BeforeEach(func() {
			var jxObjects []runtime.Object

			sourceRepo := &v1.SourceRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "jx",
					Labels:    map[string]string{"owner": "jenkins-x", "repository": "jx"},
				},

				Spec: v1.SourceRepositorySpec{
					Provider: "http://github.com",
				},
			}

			jxObjects = append(jxObjects, sourceRepo)

			jxClient := fake.NewSimpleClientset(jxObjects...)

			port, _ = getFreePort()
			testController = controller{
				path:        "/",
				bindAddress: "127.0.0.1",
				port:        port,
				jxClient:    jxClient,
				ns:          "jx",
			}

		})

		It("retrieves source URL from cluster", func() {
			url := testController.getSourceURL("jenkins-x", "jx")
			Expect(url).Should(Equal(expectedURL))
		})

		It("returns the empty string for an unknown repo", func() {
			url := testController.getSourceURL("jenkins-x", "foo")
			Expect(url).Should(BeEmpty())
		})
	})
})

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
