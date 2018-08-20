package config

import (
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentExposecontrollerHelmValues(t *testing.T) {
	t.Parallel()

	testFile, err := ioutil.ReadFile("helm_values_test.yaml")
	assert.NoError(t, err)

	a := make(map[string]string)
	a["helm.sh/hook"] = "post-install,post-upgrade"
	a["helm.sh/hook-delete-policy"] = "hook-succeeded"

	values := HelmValuesConfig{
		ExposeController: &ExposeController{},
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
