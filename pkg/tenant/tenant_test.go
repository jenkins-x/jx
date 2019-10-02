package tenant

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	gke_test "github.com/jenkins-x/jx/pkg/cloud/gke/mocks"
	"github.com/petergtz/pegomock"

	"github.com/stretchr/testify/assert"
)

const (
	projectID      = "cheese"
	cluster        = "brie"
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

	gclouder := &gke_test.MockGClouder{}
	pegomock.When(gclouder.CreateDNSZone("cheese", "cheese.wine.com")).ThenReturn("123", []string{"abc"}, nil)

	s, err := cli.GetTenantSubDomain("http://localhost", "", projectID, cluster, gclouder)

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

func TestVerifyDomainName(t *testing.T) {
	t.Parallel()
	invalidErr := "domain name %s contains invalid characters"
	lengthErr := "domain name %s has fewer than 3 or greater than 63 characters"

	domain := "wine.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "more-wine.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "wine-and-cheese.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "wine-and-cheese.tasting.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "wine123.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "wine.cheese.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "wine.cheese.rocks"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "win_e.com"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "has.two.dots"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "this.has.three.dots"
	assert.Equal(t, ValidateDomainName(domain), nil)
	domain = "now.this.has.four.dots"
	assert.Equal(t, ValidateDomainName(domain), nil)

	domain = "win?e.com"
	assert.EqualError(t, ValidateDomainName(domain), fmt.Sprintf(invalidErr, domain))
	domain = "win%e.com"
	assert.EqualError(t, ValidateDomainName(domain), fmt.Sprintf(invalidErr, domain))

	domain = "om"
	assert.EqualError(t, ValidateDomainName(domain), fmt.Sprintf(lengthErr, domain))
	domain = "some.really.long.domain.that.should.be.longer.than.the.maximum.63.characters.com"
	assert.EqualError(t, ValidateDomainName(domain), fmt.Sprintf(lengthErr, domain))
}
