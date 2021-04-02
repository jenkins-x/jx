package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

func TestPlugins(t *testing.T) {
	t.Parallel()

	list := plugins.Plugins

	for _, p := range list {
		if p.Name != "gitops" {
			continue
		}
		assert.Equal(t, "jx-gitops", p.Spec.Name, "plugin.Spec.Name")

		foundLinux := false
		foundWindows := false
		for _, b := range p.Spec.Binaries {
			if b.Goos == "Linux" && b.Goarch == "amd64" {
				foundLinux = true
				assert.Equal(t, "https://github.com/jenkins-x-plugins/jx-gitops/releases/download/v"+plugins.GitOpsVersion+"/jx-gitops-linux-amd64.tar.gz", b.URL, "URL for linux binary")
				t.Logf("found linux binary URL %s", b.URL)
			} else if b.Goos == "Windows" && b.Goarch == "amd64" {
				foundWindows = true
				assert.Equal(t, "https://github.com/jenkins-x-plugins/jx-gitops/releases/download/v"+plugins.GitOpsVersion+"/jx-gitops-windows-amd64.zip", b.URL, "URL for windows binary")
				t.Logf("found windows binary URL %s", b.URL)
			}
		}
		assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", p)
		assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", p)
	}
}

func TestOctantPlugin(t *testing.T) {
	t.Parallel()

	plugin := plugins.CreateOctantPlugin("0.16.1")

	assert.Equal(t, plugins.OctantPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.OctantPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		if b.Goos == "Linux" && b.Goarch == "amd64" {
			foundLinux = true
			assert.Equal(t, "https://github.com/vmware-tanzu/octant/releases/download/v0.16.1/octant_0.16.1_Linux-64bit.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		} else if b.Goos == "Windows" && b.Goarch == "amd64" {
			foundWindows = true
			assert.Equal(t, "https://github.com/vmware-tanzu/octant/releases/download/v0.16.1/octant_0.16.1_Windows-64bit.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		} else if b.Goos == "Darwin" {
			foundWindows = true
			assert.Equal(t, "https://github.com/vmware-tanzu/octant/releases/download/v0.16.1/octant_0.16.1_macOS-64bit.tar.gz", b.URL, "URL for macOs binary")
			t.Logf("found Darwin binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}

func TestOctantJXPlugin(t *testing.T) {
	t.Parallel()

	plugin := plugins.CreateOctantJXPlugin("0.0.31")

	assert.Equal(t, plugins.OctantJXPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.OctantJXPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		if b.Goos == "Linux" && b.Goarch == "amd64" {
			foundLinux = true
			assert.Equal(t, "https://github.com/jenkins-x-plugins/octant-jx/releases/download/v0.0.31/octant-jx-linux-amd64.tar.gz", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		} else if b.Goos == "Windows" && b.Goarch == "amd64" {
			foundWindows = true
			assert.Equal(t, "https://github.com/jenkins-x-plugins/octant-jx/releases/download/v0.0.31/octant-jx-windows-amd64.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}
