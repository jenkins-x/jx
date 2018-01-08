package jenkins

import (
	"io/ioutil"
	"testing"
	"path/filepath"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsConfig(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "jx-test-jenkins-config-")
	assertNoError(t, err)

	fileName := filepath.Join(dir, "jenkins.yaml")

	t.Logf("Using config file %s\n", fileName)

	svc := JenkinsConfigService{
		FileName: fileName,
	}

	config, err := svc.LoadConfig()
	assertNoError(t, err)

	assert.Equal(t, 0, len(config.Servers), "Should have no servers in config but got %v", config)

	config.SetAuth("http://dummy/", &JenkinsAuth{
		Username: "someone",
		ApiToken: "someToken",
	})
	err = svc.SaveConfig(&config)
	assertNoError(t, err)

	config, err = svc.LoadConfig()
	assert.Equal(t, 1, len(config.Servers), "Should have no servers in config but got %v", config)

	t.Logf("Has config %v\n", config)
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Logf("Failed with error %v", err)
		assert.Fail(t, "Should not have received an error but got: %s", err)
	}
}
