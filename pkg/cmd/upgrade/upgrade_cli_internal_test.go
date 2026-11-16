package upgrade

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionStreamURLFromKptfile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	o := CLIOptions{}

	// no local Kptfile: falls back to the latest upstream version stream
	gitURL, err := o.getVersionStreamURL("")
	assert.NoError(t, err)
	assert.Equal(t, LatestVersionstreamURL, gitURL)

	// a broken local Kptfile must be reported instead of silently falling
	// back to the default version stream
	kptfile := filepath.Join("versionStream", "Kptfile")
	assert.NoError(t, os.MkdirAll(filepath.Dir(kptfile), 0755))
	assert.NoError(t, os.WriteFile(kptfile, []byte("not: [valid"), 0644))
	_, err = o.getVersionStreamURL("")
	assert.Error(t, err)

	// a local Kptfile without an upstream git repository is also invalid
	assert.NoError(t, os.WriteFile(kptfile, []byte("apiVersion: kpt.dev/v1alpha1\nkind: Kptfile\n"), 0644))
	_, err = o.getVersionStreamURL("")
	assert.Error(t, err)

	// a valid Kptfile provides the version stream URL
	assert.NoError(t, os.WriteFile(kptfile, []byte("upstream:\n  git:\n    repo: https://example.com/my-version-stream.git\n"), 0644))
	gitURL, err = o.getVersionStreamURL("")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/my-version-stream.git", gitURL)
}
