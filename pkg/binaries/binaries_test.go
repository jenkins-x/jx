package binaries

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallIfExtractorIsEmpty(t *testing.T) {
	t.Parallel()
	shouldInstall, err := ShouldInstallBinary("cat", "0.0", nil)
	assert.True(t, shouldInstall)
	assert.Nil(t, err)
}

type mockedCatVersionExtractor struct {
}

func (mockedCatVersionExtractor) arguments() []string {
	return []string{"--version"}
}
func (mockedCatVersionExtractor) extractVersion(command string, arguments []string) (string, error) {
	return "0.0", nil
}

func TestShouldNotInstall(t *testing.T) {
	t.Parallel()
	shouldInstall, err := ShouldInstallBinary("cat", "0.0", mockedCatVersionExtractor{})
	assert.False(t, shouldInstall)
	assert.Nil(t, err)
}

func TestShouldInstall(t *testing.T) {
	t.Parallel()
	shouldInstall, err := ShouldInstallBinary("cat", "0.1", mockedCatVersionExtractor{})
	assert.True(t, shouldInstall)
	assert.Nil(t, err)
}
