package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestStepVerifyPod(t *testing.T) {
	t.Parallel()

	options := cmd.StepVerifyPodOptions{}
	// fake the output stream to be checked later
	r, fakeStdout, _ := os.Pipe()
	options.CommonOptions = cmd.CommonOptions{
		Out: fakeStdout,
		Err: os.Stderr,
	}

	cmd.ConfigureTestOptions(&options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	err := options.Run()
	assert.NoError(t, err, "Command failed: %#v", options)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	assert.Contains(t, string(outBytes), "POD STATUS")

}

func TestStepVerifyPodDebug(t *testing.T) {
	t.Parallel()

	options := cmd.StepVerifyPodOptions{Debug: true}
	// fake the output stream to be checked later
	r, fakeStdout, _ := os.Pipe()
	options.CommonOptions = cmd.CommonOptions{
		Out: fakeStdout,
		Err: os.Stderr,
	}

	cmd.ConfigureTestOptions(&options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	err := options.Run()
	assert.NoError(t, err, "Command failed: %#v", options)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	assert.Contains(t, string(outBytes), "POD STATUS")

	//check DEBUG file created
	filename := "verify-pod.log"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Debug log does not exist")
	}

	assert.NoError(t, err, "Command failed: %#v", options)

}