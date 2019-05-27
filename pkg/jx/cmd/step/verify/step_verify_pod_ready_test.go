package verify_test

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/jx/cmd/step/verify"
	"io/ioutil"
	"os"
	"testing"

	gits_test "github.com/jenkins-x/jx/pkg/gits/mocks"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/stretchr/testify/assert"
)

func TestStepVerifyPod(t *testing.T) {
	t.Parallel()

	options := verify.StepVerifyPodReadyOptions{}
	// fake the output stream to be checked later
	r, fakeStdout, _ := os.Pipe()
	commonOpts := opts.NewCommonOptionsWithFactory(nil)
	commonOpts.Out = fakeStdout
	commonOpts.Err = os.Stderr
	options.CommonOptions = &commonOpts

	cmd_test_helpers.ConfigureTestOptions(options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
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

	options := verify.StepVerifyPodReadyOptions{Debug: true}
	// fake the output stream to be checked later
	r, fakeStdout, _ := os.Pipe()
	commonOpts := opts.NewCommonOptionsWithFactory(nil)
	commonOpts.Out = fakeStdout
	commonOpts.Err = os.Stderr
	options.CommonOptions = &commonOpts

	cmd_test_helpers.ConfigureTestOptions(options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
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
