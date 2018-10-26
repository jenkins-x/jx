package config_test

import (
	"github.com/jenkins-x/jx/pkg/tests"
	"testing"

	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentExposecontrollerHelmValues(t *testing.T) {
	tests.SkipForWindows(t, "Pre-existing test. Reason not investigated")
	t.Parallel()

	testFile, err := ioutil.ReadFile("helm_values_test.yaml")
	assert.NoError(t, err)

	a := make(map[string]string)
	a["helm.sh/hook"] = "post-install,post-upgrade"
	a["helm.sh/hook-delete-policy"] = "hook-succeeded"

	values := config.HelmValuesConfig{
		ExposeController: &config.ExposeController{},
	}

	values.ExposeController.Annotations = a
	values.ExposeController.Config.Exposer = "Ingress"
	values.ExposeController.Config.Domain = "jenkinsx.io"
	values.ExposeController.Config.HTTP = "false"
	values.ExposeController.Config.TLSAcme = "false"
	s, err := values.String()
	assert.NoError(t, err)
	assert.Equal(t, string(testFile), s, "expected exposecontroller helm values do not match")
}
