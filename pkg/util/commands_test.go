package util_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestRunPass(t *testing.T) {

	tmpFileName := "test_run_pass.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "5"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name: ex,
		Dir:  exPath,
		Args: args,
	}

	res, err := cmd.Run()

	assert.NoError(t, err, "Run should exit without failure")
	assert.Equal(t, "PASS", res)
	assert.Equal(t, 4, len(cmd.Errors))
	assert.Equal(t, 5, cmd.Attempts())
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, false, cmd.DidFail())
	assert.NotEqual(t, nil, cmd.Error())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunPassFirstTime(t *testing.T) {

	tmpFileName := "test_run_pass_first_time.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
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

	tmpFileName := "test_run_fail_with_timeout.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
	}

	res, err := cmd.Run()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunThreadSafety(t *testing.T) {
	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
	ex := "sleep.sh"
	args := []string{"2"}

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 1 * time.Second,
	}

	res, err := cmd.Run()

	assert.NoError(t, err, "Run should exit without failure")
	assert.Equal(t, "2", res)
	assert.Equal(t, false, cmd.DidError())
	assert.Equal(t, false, cmd.DidFail())
	assert.Equal(t, 1, cmd.Attempts())
}

func TestRunWithoutRetry(t *testing.T) {

	tmpFileName := "test_run_without_retry.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
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
	assert.Equal(t, false, cmd.Verbose)
	assert.Equal(t, false, cmd.Quiet)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunVerbose(t *testing.T) {

	tmpFileName := "test_run_verbose.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
		Verbose: true,
	}

	res, err := cmd.RunWithoutRetry()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "FAILURE!", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())
	assert.Equal(t, true, cmd.IsVerbose())
	assert.Equal(t, false, cmd.IsQuiet())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunQuiet(t *testing.T) {

	tmpFileName := "test_run_quiet.txt"

	gp := os.Getenv("GOPATH")
	projectRoot := path.Join(gp, "src/github.com/jenkins-x/jx")
	exPath := projectRoot + "/pkg/jx/cmd/test_data/scripts"
	ex := "fail_iterator.sh"
	args := []string{tmpFileName, "100"}

	os.Create(exPath + "/" + tmpFileName)

	cmd := util.Command{
		Name:    ex,
		Dir:     exPath,
		Args:    args,
		Timeout: 3 * time.Second,
		Quiet:   true,
	}

	res, err := cmd.RunWithoutRetry()

	assert.Error(t, err, "Run should exit with failure")
	assert.Equal(t, "", res)
	assert.Equal(t, true, cmd.DidError())
	assert.Equal(t, true, cmd.DidFail())
	assert.Equal(t, 1, len(cmd.Errors))
	assert.Equal(t, 1, cmd.Attempts())
	assert.Equal(t, false, cmd.IsVerbose())
	assert.Equal(t, true, cmd.IsQuiet())

	os.Remove(exPath + "/" + tmpFileName)

}

func TestRunIsVerboseAndIsQuiet(t *testing.T) {

	cmd := util.Command{}
	assert.Equal(t, false, cmd.IsVerbose())
	assert.Equal(t, false, cmd.IsQuiet())

	cmd = util.Command{
		Verbose: true,
	}
	assert.Equal(t, true, cmd.IsVerbose())
	assert.Equal(t, false, cmd.IsQuiet())

	cmd = util.Command{
		Verbose: true,
		Quiet:   true,
	}
	assert.Equal(t, true, cmd.IsVerbose())
	assert.Equal(t, false, cmd.IsQuiet())

	cmd = util.Command{
		Quiet: true,
	}
	assert.Equal(t, false, cmd.IsVerbose())
	assert.Equal(t, true, cmd.IsQuiet())

}
