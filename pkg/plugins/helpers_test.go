package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx-cli/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

func TestPlugins(t *testing.T) {
	t.Parallel()

	list := plugins.Plugins

	for _, p := range list {
		if p.Name == "gitops" {
			assert.Equal(t, "jx-gitops", p.Spec.Name, "plugin.Spec.Name")

			foundLinux := false
			foundWindows := false
			for _, b := range p.Spec.Binaries {
				if b.Goos == "Linux" && b.Goarch == "amd64" {
					foundLinux = true
					assert.Equal(t, "https://storage.googleapis.com/cloudbees-jx-plugins/plugin/gitops/"+plugins.GitOpsVersion+"/jx-gitops-linux-amd64.tar.gz", b.URL, "URL for linux binary")
					t.Logf("found linux binary URL %s", b.URL)
				} else if b.Goos == "Windows" && b.Goarch == "amd64" {
					foundWindows = true
					assert.Equal(t, "https://storage.googleapis.com/cloudbees-jx-plugins/plugin/gitops/"+plugins.GitOpsVersion+"/jx-gitops-windows-amd64.zip", b.URL, "URL for windows binary")
					t.Logf("found windows binary URL %s", b.URL)
				}
			}
			assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", p)
			assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", p)
		}
	}
}
