// +build unit

package verify

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	versionstream_test "github.com/jenkins-x/jx/v2/pkg/versionstream/mocks"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
)

const upgradeMessage = "jx upgrade cli"

var compareTests = []struct {
	description      string
	directory        string
	shouldUpgradeCli bool
}{
	{"version stream has greater version", "test_data/verify_packages/greater", true},
	{"version stream has same version", "test_data/verify_packages/same", false},
	{"version stream is outdated and has lesser version", "test_data/verify_packages/lesser", false},
}

func TestStepVerifyPackageOptions_VerifyJXVersion(t *testing.T) {

	//Create a mock resolver
	resolver := versionstream_test.NewMockStreamer()
	for _, tt := range compareTests {
		t.Run(tt.description, func(t *testing.T) {
			out := &testhelpers.FakeOut{}
			options := opts.NewCommonOptionsWithTerm(fake.NewFakeFactory(), os.Stdin, out, os.Stderr)

			options.BatchMode = true

			stepOptions := step.StepOptions{options, false, ""}
			stepVerifyPackagesOptions := StepVerifyPackagesOptions{stepOptions, "", false, nil, ""}
			pegomock.When(resolver.GetVersionsDir()).ThenReturn(tt.directory)
			log.SetOutput(out)
			err := stepVerifyPackagesOptions.verifyJXVersion(resolver)
			assert.NoError(t, err)
			if tt.shouldUpgradeCli {
				assert.Contains(t, out.GetOutput(), upgradeMessage)
			} else {
				assert.NotContains(t, out.GetOutput(), upgradeMessage)
			}
		})
	}
}
