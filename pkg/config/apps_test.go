// +build unit

package config

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsXAppsUnmarshalling(t *testing.T) {
	apps, _, err := LoadAppConfig(path.Join("test_data"))
	assert.NoError(t, err)

	// assert marshalling of a jx-apps.yaml
	assert.Equal(t, 4, len(apps.Apps))
	assert.Equal(t, "cert-manager", apps.Apps[3].Namespace)
}

func TestBadPhase(t *testing.T) {
	_, _, err := LoadAppConfig(path.Join("test_data", "jx-apps-phase-bad"))
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "failed to validate YAML file"))
}

func TestGoodPhase(t *testing.T) {
	apps, _, err := LoadAppConfig(path.Join("test_data", "jx-apps-phase-good"))
	assert.NoError(t, err)
	assert.Equal(t, "velero", apps.Apps[0].Name)
	assert.Equal(t, PhaseSystem, apps.Apps[0].Phase)
	assert.Equal(t, "external-dns", apps.Apps[1].Name)
	assert.Equal(t, PhaseApps, apps.Apps[1].Phase)
}
