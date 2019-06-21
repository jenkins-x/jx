package auth

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	url1 = "http://dummy/"
	url2 = "http://another-jenkins/"

	userDoesNotExist = "doesNotExist"
	user1            = "someone"
	user2            = "another"
	token2v2         = "tokenV2"
)

func TestAuthConfig(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "jx-test-jenkins-config-")
	assertNoError(t, err)

	fileName := filepath.Join(dir, "jenkins.yaml")

	t.Logf("Using config file %s\n", fileName)

	configTest := ConfigTest{
		t: t,
	}
	configTest.svc, err = NewFileConfigService(fileName)
	assert.NoError(t, err, "should create file auth service")

	config := configTest.Config()

	assert.Equal(t, 0, len(config.Servers), "Should have no servers in config but got %v", config)
	assertNoAuth(t, config, url1, userDoesNotExist)

	auth1 := User{
		Username: user1,
		ApiToken: "someToken",
	}
	config = configTest.SetUserAuth(url1, auth1)

	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 1, len(config.Servers[0].Users), "Number of auths")
	foundAuth, err := config.GetUser(url1, user1)
	assert.NoError(t, err)
	assert.Equal(t, auth1, foundAuth, "loaded auth for server %s and user %s", url1, user1)
	_, err = config.GetUser(url1, "")
	assert.Error(t, err)

	auth2 := User{
		Username: user2,
		ApiToken: "anotherToken",
	}
	config = configTest.SetUserAuth(url1, auth2)

	foundAuth2, err := config.GetUser(url1, user2)
	assert.NoError(t, err)
	assert.Equal(t, auth2, foundAuth2, "Failed to find auth for server %s and user %s", url1, user2)
	foundAuth1, err := config.GetUser(url1, user1)
	assert.NoError(t, err)
	assert.Equal(t, auth1, foundAuth1, "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths")
	assertNoAuth(t, config, url1, userDoesNotExist)

	// lets mutate the auth2
	auth2.ApiToken = token2v2
	config = configTest.SetUserAuth(url1, auth2)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths")
	foundUser1, err := config.GetUser(url1, user1)
	assert.NoError(t, err)
	assert.Equal(t, auth1, foundUser1, "loaded auth for server %s and user %s", url1, user1)
	foundUser2, err := config.GetUser(url1, user2)
	assert.NoError(t, err)
	assert.Equal(t, auth2, foundUser2, "loaded auth for server %s and user %s", url1, user2)

	auth3 := User{
		Username: user1,
		ApiToken: "server2User1Token",
	}
	configTest.SetUserAuth(url2, auth3)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 2, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths for server 0")
	assert.Equal(t, 1, len(config.Servers[1].Users), "Number of auths for server 1")
	foundUser1, err = config.GetUser(url1, user1)
	assert.NoError(t, err)
	assert.Equal(t, auth1, foundUser1, "loaded auth for server %s and user %s", url1, user1)
	foundUser2, err = config.GetUser(url1, user2)
	assert.NoError(t, err)
	assert.Equal(t, auth2, foundUser2, "loaded auth for server %s and user %s", url1, user2)
	foundUser3, err := config.GetUser(url2, user1)
	assert.NoError(t, err)
	assert.Equal(t, auth3, foundUser3, "loaded auth for server %s and user %s", url2, user1)
}

type ConfigTest struct {
	t   *testing.T
	svc ConfigService
}

func (c *ConfigTest) SetUserAuth(url string, auth User) *Config {
	config, err := c.svc.Config()
	c.AssertNoError(err)
	config.AddServer(url, "", "", false)
	err = config.AddUserToServer(url, auth)
	c.AssertNoError(err)
	return c.SaveAndReload()
}

func (c *ConfigTest) SaveAndReload() *Config {
	err := c.svc.SaveConfig()
	c.AssertNoError(err)
	err = c.svc.LoadConfig()
	c.AssertNoError(err)
	config, err := c.svc.Config()
	c.AssertNoError(err)
	return config
}

func (c *ConfigTest) Config() *Config {
	config, err := c.svc.Config()
	c.AssertNoError(err)
	return config
}

func (c *ConfigTest) AssertNoError(err error) {
	if err != nil {
		assert.Fail(c.t, "Should not have received an error but got: %s", err)
	}
}

func assertNoAuth(t *testing.T, config *Config, url string, user string) {
	_, err := config.GetUser(url, user)
	if err == nil {
		assert.Fail(t, "Found auth when not expecting it for server %s and user %s", url, user)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		assert.Fail(t, "Should not have received an error but got: %s", err)
	}
}

func TestAuthConfigGetsDefaultName(t *testing.T) {
	t.Parallel()
	c := &Config{}

	expectedURL := "https://foo.com"
	c.AddServer(expectedURL, "foo", "", false)
	server, err := c.GetServer(expectedURL)
	assert.NoError(t, err)
	assert.NotNil(t, server, "No server found!")
	assert.True(t, server.Name != "", "Should have a server name!")
	assert.Equal(t, expectedURL, server.URL, "Server.URL")
}

func TestDeleteServer(t *testing.T) {
	t.Parallel()
	c := &Config{}
	url := "https://foo.com"
	c.AddServer(url, "", "", false)
	server, err := c.GetServer(url)
	assert.NoError(t, err)
	assert.NotNil(t, server, "Failed to add the server to the configuration")
	assert.Equal(t, 1, len(c.Servers), "No server found in the configuration")

	c.DeleteServer(url)
	assert.Equal(t, 0, len(c.Servers), "Failed to remove the server from configuration")
	assert.Equal(t, "", c.CurrentServer, "Should be no current server")
}

func TestDeleteServer2(t *testing.T) {
	t.Parallel()
	c := &Config{}
	url1 := "https://foo1.com"
	c.AddServer(url1, "", "", false)
	server1, err := c.GetServer(url1)
	assert.NoError(t, err)
	assert.NotNil(t, server1, "Failed to add the server to the configuration")
	url2 := "https://foo2.com"
	c.AddServer(url2, "", "", false)
	server2, err := c.GetServer(url2)
	assert.NoError(t, err)
	assert.NotNil(t, server2, "Failed to the server to the configuration!")
	assert.Equal(t, 2, len(c.Servers), "Must have 2 servers in the configuration")
	c.CurrentServer = url2

	c.DeleteServer(url2)
	assert.Equal(t, 1, len(c.Servers), "Failed to remove one server from configuration")
	assert.Equal(t, url1, c.Servers[0].URL, "Failed to remove the right server from the configuration")
	assert.Equal(t, url1, c.CurrentServer, "Server 1 should be current server")
}
