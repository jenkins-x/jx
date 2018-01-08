package jenkins

import (
	"io/ioutil"
	"testing"
	"path/filepath"

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

func TestJenkinsConfig(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "jx-test-jenkins-config-")
	assertNoError(t, err)

	fileName := filepath.Join(dir, "jenkins.yaml")

	t.Logf("Using config file %s\n", fileName)

	configTest := ConfigTest{
		t: t,
	}
	configTest.svc.FileName = fileName

	config := configTest.Load()

	assert.Equal(t, 0, len(config.Servers), "Should have no servers in config but got %v", config)
	assertNoAuth(t, config, url1, userDoesNotExist)

	auth1 := JenkinsAuth{
		Username: user1,
		ApiToken: "someToken",
	}
	config = configTest.SetAuth(url1, auth1)

	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 1, len(config.Servers[0].Auths), "Number of auths")
	assert.Equal(t, &auth1, config.FindAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth1, config.FindAuth(url1, ""), "loaded auth for server %s and no user", url1)

	auth2 := JenkinsAuth{
		Username: user2,
		ApiToken: "anotherToken",
	}
	config = configTest.SetAuth(url1, auth2)

	assert.Equal(t, &auth2, config.FindAuth(url1, user2), "Failed to find auth for server %s and user %s", url1, user2)
	assert.Equal(t, &auth1, config.FindAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Auths), "Number of auths")
	assertNoAuth(t, config, url1, userDoesNotExist)

	// lets mutate the auth2
	auth2.ApiToken = token2v2
	config = configTest.SetAuth(url1, auth2)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Auths), "Number of auths")
	assert.Equal(t, &auth1, config.FindAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth2, config.FindAuth(url1, user2), "loaded auth for server %s and user %s", url1, user2)

	auth3 := JenkinsAuth{
		Username: user1,
		ApiToken: "server2User1Token",
	}
	configTest.SetAuth(url2, auth3)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 2, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Auths), "Number of auths for server 0")
	assert.Equal(t, 1, len(config.Servers[1].Auths), "Number of auths for server 1")
	assert.Equal(t, &auth1, config.FindAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth2, config.FindAuth(url1, user2), "loaded auth for server %s and user %s", url1, user2)
	assert.Equal(t, &auth3, config.FindAuth(url2, user1), "loaded auth for server %s and user %s", url2, user1)
}

type ConfigTest struct {
	t      *testing.T
	svc    JenkinsConfigService
	config JenkinsConfig
}

func (c *ConfigTest) Load() *JenkinsConfig  {
	config, err := c.svc.LoadConfig()
	c.config = config
	c.AssertNoError(err)
	return &c.config
}

func (c *ConfigTest) SetAuth(url string, auth JenkinsAuth) *JenkinsConfig {
	c.config.SetAuth(url, auth)
	c.SaveAndReload()
	return &c.config
}

func (c *ConfigTest) SaveAndReload() *JenkinsConfig {
	err := c.svc.SaveConfig(&c.config)
	c.AssertNoError(err)
	return c.Load()
}

func (c *ConfigTest) AssertNoError(err error) {
	if err != nil {
		assert.Fail(c.t, "Should not have received an error but got: %s", err)
	}
}

func assertNoAuth(t *testing.T, config *JenkinsConfig, url string, user string) {
	found := config.FindAuth(url, user)
	if found != nil {
		assert.Fail(t, "Found auth when not expecting it for server %s and user %s", url, user)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		assert.Fail(t, "Should not have received an error but got: %s", err)
	}
}
