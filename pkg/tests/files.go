package tests

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

// AssertFileContains asserts that a given file exists and contains the given text
func AssertFileContains(t *testing.T, fileName string, containsText string) {
	if AssertFileExists(t, fileName) {
		data, err := ioutil.ReadFile(fileName)
		assert.NoError(t, err, "Failed to read file %s", fileName)
		if err == nil {
			text := string(data)
			assert.True(t, strings.Index(text, containsText) >= 0, "The file %s does not contain text: %s", fileName, containsText)
		}
	}
}

// AssertFileExists asserts that the given file exists
func AssertFileExists(t *testing.T, fileName string) bool {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.True(t, exists, "File %s should exist", fileName)
	if exists {
		Debugf("File %s exists\n", fileName)
	}
	return exists
}

// AssertFileDoesNotExist asserts that the given file does not exist
func AssertFileDoesNotExist(t *testing.T, fileName string) bool {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.False(t, exists, "File %s should not exist", fileName)
	if exists {
		Debugf("File %s does not exist\n", fileName)
	}
	return exists
}

// AssertFilesExist asserts that the list of file paths either exist or don't exist
func AssertFilesExist(t *testing.T, expected bool, paths ...string) {
	for _, path := range paths {
		if expected {
			AssertFileExists(t, path)
		} else {
			AssertFileDoesNotExist(t, path)
		}
	}
}
