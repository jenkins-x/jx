package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestStepSplitMonorepo(t *testing.T) {
	testData := filepath.Join("test_data", "split_monorepo")

	tempDir, err := ioutil.TempDir("", "test_split_monorepo")
	assert.NoError(t, err)

	options := &StepSplitMonorepoOptions{
		Organisation: "dummy",
		Glob:         "*",
		Dir:          testData,
		OutputDir:    tempDir,
		NoGit:        true,
	}

	err = options.Run()
	assert.NoError(t, err, "Failed to run split monorepo on source %s output %s", testData, tempDir)

	tests.Debugf("Generated split repos in: %s\n", tempDir)
	fmt.Printf("Generated split repos in: %s\n", tempDir)

	assertFilesExist(t, true, filepath.Join(tempDir, "bar"), filepath.Join(tempDir, "foo"))
	assertFilesExist(t, false, filepath.Join(tempDir, "kubernetes"))

	assertFilesExist(t, true,
		filepath.Join(tempDir, "bar", "charts", "bar", "Chart.yaml"),
		filepath.Join(tempDir, "bar", "charts", "bar", "templates", "bar-deployment.yaml"))
}

func assertFilesExist(t *testing.T, expected bool, paths ...string) {
	for _, path := range paths {
		if expected {
			assertFileExists(t, path)
		} else {
			assertFileDoesNotExist(t, path)
		}
	}
}
