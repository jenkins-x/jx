// +build soak

package buildnum

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	. "github.com/petergtz/pegomock"

	build_num_test "github.com/jenkins-x/jx/pkg/buildnum/mocks"
	"github.com/jenkins-x/jx/pkg/buildnum/mocks/matchers"

	"github.com/stretchr/testify/assert"
)

const (
	numRequests = 5000
	numThreads  = 25
)

func TestSoakTest(t *testing.T) {
	mockIssuer := build_num_test.NewMockBuildNumberIssuer()
	When(mockIssuer.NextBuildNumber(matchers.AnyKubePipelineID())).Then(generateBuildNumber)
	server := NewHTTPBuildNumberServer("", 1234, mockIssuer)

	fmt.Printf("Using %d clients to make %d requests each.\n", numThreads, numRequests)

	wg := sync.WaitGroup{}

	for i := 0; i < numThreads; i++ {
		go func() {
			wg.Add(1)
			for i := 0; i < numRequests; i++ {
				runVendClient(t, server)
			}
			wg.Done()
		}()
	}

	fmt.Printf("Waiting for %d clients to complete.\n", numThreads)
	wg.Wait()
	fmt.Print("Done\n")
}

// To be called by pegomock as NextBuildNumber() implementation - add a small sleep to simulate the real processing &
// increase concurrency in the build number service.
func generateBuildNumber(params []Param) ReturnValues {
	time.Sleep(time.Millisecond * 25)
	return ReturnValues{strconv.Itoa(rand.Intn(numThreads)), nil}
}

// Act as a client making a /vend request to the specified server.
func runVendClient(t *testing.T, server *HTTPBuildNumberServer) {
	//pID := kube.NewPipelineIDFromString(fmt.Sprintf("owner1/repo1/branch-%d", rand.Intn(100)))
	//pegomock.When(mockIssuer.NextBuildNumber(matchers.EqKubePipelineID(pID))).ThenReturn("3", nil)

	path := fmt.Sprintf("owner1/repo1/branch-%d", rand.Intn(25))
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal("Unexpected error setting up fake HTTP request.", err)
	}

	rr := httptest.NewRecorder()
	server.vend(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Expected OK status code for valid /vend GET request.")
}
