package util_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestRunPass(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_pass.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "3"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 15 * time.Second,
	}

	res, err := cmd.Run()

	assert.NoError(t, err, "Run should exit without failure")
	assert.Equal(t, "PASS", res)
	assert.Equal(t, 2, len(cmd.Errors))
	assert.Equal(t, 3, cmd.Attempts())
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, false, cmd.DidFail())
	assert.NotEqual(t, nil, cmd.Error())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunPassFirstTime(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_pass_first_time.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "1"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name: ex,
		Dir:  exPath,
		Args: args,
	}

	res, err := cmd.Run()

	assert.NoError(t, err, "Run should exit without failure")
	assert.Equal(t, "PASS", res)
	assert.Equal(t, 0, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())
	assert.Equal(t, false, cmd.DidError())
	assert.Equal(t, false, cmd.DidFail())
	assert.Equal(t, nil, cmd.Error())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunFailWithTimeout(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_fail_with_timeout.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 1 * time.Second,
	}

	res, err := cmd.Run()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunThreadSafety(t *testing.T) {
	t.Parallel()
	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "sleep.sh"
	args := []string{"0.2"}

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 10000000 * time.Nanosecond,
	}

	res, err := cmd.Run()

	assert.NoError(t, err, "Run should exit without failure")
	assert.Equal(t, "0.2", res)
	assert.Equal(t, false, cmd.DidError())
	assert.Equal(t, false, cmd.DidFail())
	assert.Equal(t, 1, cmd.Attempts())
}

func TestRunWithoutRetry(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_without_retry.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
	}

	res, err := cmd.RunWithoutRetry()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "FAILURE!", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunVerbose(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_verbose.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
	}

	res, err := cmd.RunWithoutRetry()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "FAILURE!", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunQuiet(t *testing.T) {
	t.Parallel()

	tmpFileName := "test_run_quiet.txt"

	startPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}
	exPath := startPath + "/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
	}

	res, err := cmd.RunWithoutRetry()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "FAILURE!", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())

	os.Remove(exPath + "/" + tmpFileName)

}
