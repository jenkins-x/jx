package binaries

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallIfExtractorIsEmpty(t *testing.T) {
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
	shouldInstall, err := ShouldInstallBinary("cat", "0.0", mockedCatVersionExtractor{})
	assert.False(t, shouldInstall)
	assert.Nil(t, err)
}

func TestShouldInstall(t *testing.T) {
	shouldInstall, err := ShouldInstallBinary("cat", "0.1", mockedCatVersionExtractor{})
	assert.True(t, shouldInstall)
	assert.Nil(t, err)
}

// Test extracting version from cat

type catVersionExtractor struct {
}

func (catVersionExtractor) arguments() []string {
	return []string{"--version"}
}
func (catVersionExtractor) extractVersion(command string, arguments []string) (string, error) {
	cmd := exec.Command(command, arguments...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.Split(string(output), "\n")[0], nil
}

func TestShouldInstallWhenVersionIsMismatched(t *testing.T) {
	shouldInstall, err := ShouldInstallBinary("cat", "0.1", catVersionExtractor{})
	assert.True(t, shouldInstall)
	assert.Nil(t, err)
}
