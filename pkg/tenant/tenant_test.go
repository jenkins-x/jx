package tenant

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	projectID      = "cheese"
	domain         = "wine.com"
	subDomain      = projectID + "." + domain
	zone           = "zone"
	domainResponse = `{
		"data": {
			"subdomain": "cheese.wine.com"
		}
	}`
	nameServersResponse = `{
		"data": {
			"message": "nameServers registered"
		}
	}`
)

func TestClientGetTenantSubDomain(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(domainResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	cli := NewTenantClient()
	cli.httpClient = httpClient

	s, err := cli.GetTenantSubDomain("http://localhost", "", projectID)

	assert.Nil(t, err)
	assert.Equal(t, subDomain, s)
}

func TestClientPostTenantZoneNameServers(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nameServersResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	cli := NewTenantClient()
	cli.httpClient = httpClient

	nameServers := []string{"nameServer1", "nameServer2"}
	err := cli.PostTenantZoneNameServers("http://localhost", "", projectID, subDomain, zone, nameServers)
	assert.Nil(t, err)
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	h := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, h.Listener.Addr().String())
			},
		},
	}

	return cli, h.Close
}

func TestGetBasicAuthUserAndPassword(t *testing.T) {
	auth := "some_user:some_password"
	user, pass := getBasicAuthUserAndPassword(auth)
	assert.Equal(t, auth, user+":"+pass)
}
