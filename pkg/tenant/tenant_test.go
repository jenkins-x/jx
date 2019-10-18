package tenant

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	fake "github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/stretchr/testify/require"

	gkeTest "github.com/jenkins-x/jx/pkg/cloud/gke/mocks"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
)

const (
	projectID               = "cheese"
	cluster                 = "brie"
	domain                  = "wine.com"
	subDomain               = projectID + "." + domain
	zone                    = "zone"
	secretName              = "name"
	secretKey               = "key"
	tempToken               = "a_temporary_test_token"
	getTokenResponse        = "a_real_test_token"
	deleteTempTokenResponse = "temporary token deleted"
	domainResponse          = `{
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

func TestClientGetAndStoreTenantToken(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(getTokenResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Namespace = namespace

	err = tenant.GetAndStoreTenantToken("http://localhost", "", projectID, tempToken)
	assert.Nil(t, err)
}

func TestClientGetTenantToken(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(getTokenResponse))
	})

	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	token, err := tenant.getTenantToken("http://localhost", "", projectID, tempToken)
	assert.Nil(t, err)
	assert.Equal(t, getTokenResponse, token)
}

func TestClientDeleteTempTenantToken(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(deleteTempTokenResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Namespace = namespace

	response, err := tenant.deleteTempTenantToken("http://localhost", "", projectID, tempToken)
	assert.Nil(t, err)
	assert.Equal(t, deleteTempTokenResponse, response)
}

func TestClientWriteKubernetesSecret(t *testing.T) {
	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.Kube = client
	tenant.Namespace = namespace

	err = tenant.writeKubernetesSecret(secretName, secretKey, []byte(getTokenResponse))
	assert.Nil(t, err)
}

func TestClientGetTenantSubDomain(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(domainResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	gclouder := &gkeTest.MockGClouder{}
	pegomock.When(gclouder.CreateDNSZone("cheese", "cheese.wine.com")).ThenReturn("123", []string{"abc"}, nil)

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Gcloud = gclouder
	tenant.Namespace = namespace
	s, err := tenant.GetTenantSubDomain("http://localhost", "", projectID, cluster)

	assert.Nil(t, err)
	assert.Equal(t, subDomain, s)
}

func TestClientGetTenantSubDomainwithoutProjectID(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(domainResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	gclouder := &gkeTest.MockGClouder{}
	pegomock.When(gclouder.CreateDNSZone("cheese", "cheese.wine.com")).ThenReturn("123", []string{"abc"}, nil)

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Gcloud = gclouder
	tenant.Namespace = namespace
	_, err = tenant.GetTenantSubDomain("http://localhost", "", "", cluster)

	assert.NotNil(t, err)
	assert.EqualError(t, err, "projectID is empty")
}

func TestClientPostTenantZoneNameServers(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nameServersResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Namespace = namespace
	nameServers := []string{"nameServer1", "nameServer2"}
	err = tenant.PostTenantZoneNameServers("http://localhost", "", projectID, subDomain, zone, nameServers)
	assert.Nil(t, err)
}

func TestClientPostTenantZoneNameServersWithEmptyNameServers(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nameServersResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Namespace = namespace
	nameServers := []string{}
	err = tenant.PostTenantZoneNameServers("http://localhost", "", projectID, subDomain, zone, nameServers)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "projectID/zone/nameServers is empty")
}

func TestClientPostTenantZoneNameServersWithEmptyProject(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nameServersResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	f := fake.NewFakeFactory()
	client, namespace, err := f.CreateKubeClient()
	require.NoError(t, err, "CreateKubeClient() failed")
	assert.Equal(t, "jx", namespace, "namespace")
	assert.NotNil(t, client, "client")

	tenant := NewTenantClient()
	tenant.HttpClient = httpClient
	tenant.Kube = client
	tenant.Namespace = namespace
	nameServers := []string{"nameServer1", "nameServer2"}
	err = tenant.PostTenantZoneNameServers("http://localhost", "", "", subDomain, zone, nameServers)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "projectID/zone/nameServers is empty")
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
	user, pass := util.GetBasicAuthUserAndPassword(auth)
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
