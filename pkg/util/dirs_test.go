package util

import (
	"errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJXBinaryLocationSuccess(t *testing.T) {
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "[TEST SETUP] failed to create temp directory for test")
	defer os.RemoveAll(tempDir)
	jxpath := filepath.Join(tempDir, "jx")
	// resolving symlinks requires that the file exists (at least on windows...
	_ , err = os.Create(jxpath)
	require.NoError(t, err, "[TEST SETUP] failed to create temp directory for test")

	res, err := jXBinaryLocation(stubFunction(jxpath, nil))
	assert.Equal(t, tempDir, res)
	assert.NoError(t, err, "Should not error")
}

func TestJXBinaryLocationFailure(t *testing.T) {
	t.Parallel()

	res, err := jXBinaryLocation(stubFunction("", errors.New("Kaboom")))
	assert.Equal(t, "", res)
	assert.Error(t, err, "Should error")
}

func stubFunction(str string, err error) func () (string, error) {
	return func () (string, error) {
		return str, err
	}
}
