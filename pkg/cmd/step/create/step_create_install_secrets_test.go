package create_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallSecrets(t *testing.T) {
	testData := path.Join("test_data", "step_create_install_secrets")
	assert.DirExists(t, testData)

	outputDir, err := ioutil.TempDir("", "test-step-create-install-secrets-")
	require.NoError(t, err)

	err = util.CopyDir(testData, outputDir, true)
	require.NoError(t, err, "failed to copy test data into temp dir")

	o := &create.StepCreateInstallSecretsOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: &opts.CommonOptions{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			},
		},
		Dir: outputDir,
	}
	err = o.Run()
	require.NoError(t, err, "failed to run step")

	tests.AssertTextFileContentsEqual(t, filepath.Join(outputDir, "secrets.yaml"), filepath.Join(outputDir, "expected-secrets.yaml"))
}
