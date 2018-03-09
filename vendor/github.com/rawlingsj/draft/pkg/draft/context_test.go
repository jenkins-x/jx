package draft

import (
	"io/ioutil"
	"strings"
	"testing"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/Azure/draft/pkg/rpc"
)

var expectedVals = `basedomain: ""
image:
  name: ""
  org: ""
  registry: ""
  tag: da39a3ee5e6b4b0d3255bfef95601890afd80709`

func newTestServer() *Server {
	cfg := &ServerConfig{
		Registry: new(RegistryConfig),
	}
	return NewServer(cfg)
}

func newTestUpRequest() *rpc.UpRequest {
	return &rpc.UpRequest{
		AppArchive: new(rpc.AppArchive),
		Values:     new(chart.Config),
	}
}

func TestNewAppContext(t *testing.T) {
	appContext, err := newAppContext(newTestServer(), newTestUpRequest(), ioutil.Discard)
	if err != nil {
		t.Errorf("expected newAppContext() with empty values should return no err, got %v", err)
	}

	vals, err := appContext.vals.YAML()
	if err != nil {
		t.Errorf("expected appContext.vals.YAML() to return no err, got %v", err)
	}
	if strings.Compare(vals, expectedVals) == 0 {
		t.Errorf("expected app context vals to have injected values\nWanted:\n%s\n---\nGot:\n%s", expectedVals, vals)
	}
}
