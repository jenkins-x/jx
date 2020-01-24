package config

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsXAppsUnmarshalling(t *testing.T) {
	apps, err := LoadApplicationsConfig(path.Join("test_data"))
	assert.NoError(t, err)

	// assert marshalling of a jx-apps.yaml
	assert.Equal(t, 4, len(apps.Applications))
	assert.Equal(t, "cert-manager", apps.Applications[3].Namespace)
}
