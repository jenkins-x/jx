// +build integration

package verify_test

import (
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/tests"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/verify"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	gits_test "github.com/jenkins-x/jx/pkg/gits/mocks"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	testDeployNamespace = "new-jx-ns"
)

func TestStepVerifyPreInstallTerraformKaniko(t *testing.T) {
	tests.Retry(t, 5, time.Second*10, func(r *tests.R) {
		options := createTestStepVerifyPreInstallOptions(filepath.Join("test_data", "preinstall", "terraform_kaniko"))

		_, origNamespace, err := options.KubeClientAndDevNamespace()
		assert.NoError(t, err)
		defer resetNamespace(t, origNamespace)

		err = options.Run()
		assert.Errorf(r, err, "the command should have failed for terraform and kaniko with a missing kaniko secret")
	})
}

func TestStepVerifyPreInstallNoKanikoNoLazyCreate(t *testing.T) {
	// TODO the fake k8s client always seems to lazily create a namespace on demand so the 'jx step verify preinstall' never fails
	t.SkipNow()

	options := createTestStepVerifyPreInstallOptions(filepath.Join("test_data", "preinstall", "no_kaniko_or_terraform"))
	// explicitly disable lazy create
	options.LazyCreateFlag = "false"

	_, origNamespace, err := options.KubeClientAndDevNamespace()
	assert.NoError(t, err)
	defer resetNamespace(t, origNamespace)

	err = options.Run()
	if err != nil {
		t.Logf("returned error: %s", err.Error())
	}
	assert.Errorf(t, err, "the command should have failed due to missing namespace")
}

func TestStepVerifyPreInstallNoKanikoLazyCreate(t *testing.T) {
	options := createTestStepVerifyPreInstallOptions(filepath.Join("test_data", "preinstall", "no_kaniko_or_terraform"))

	_, origNamespace, err := options.KubeClientAndDevNamespace()
	assert.NoError(t, err)
	defer resetNamespace(t, origNamespace)

	// we default to lazy create if not using terraform
	err = options.Run()
	assert.NoErrorf(t, err, "the command should not have failed as we should have lazily created the deploy namespace")
}

func createTestStepVerifyPreInstallOptions(dir string) *verify.StepVerifyPreInstallOptions {
	options := &verify.StepVerifyPreInstallOptions{}
	// fake the output stream to be checked later
	commonOpts := opts.NewCommonOptionsWithFactory(nil)
	options.CommonOptions = &commonOpts
	testhelpers.ConfigureTestOptions(options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	testhelpers.SetFakeFactoryFromKubeClients(options.CommonOptions)
	options.Dir = dir
	options.Namespace = testDeployNamespace
	options.Err = os.Stdout
	options.Out = os.Stdout
	return options
}

func resetNamespace(t *testing.T, ns string) {
	commonOpts := opts.NewCommonOptionsWithFactory(clients.NewFactory())
	commonOpts.Out = os.Stdout
	namespaceOptions := &namespace.NamespaceOptions{
		CommonOptions: &commonOpts,
	}
	namespaceOptions.Args = []string{ns}
	err := namespaceOptions.Run()
	assert.NoError(t, err)
}
