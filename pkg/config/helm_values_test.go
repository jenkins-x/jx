package config

import (
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentExposecontrollerHelmValues(t *testing.T) {

	testFile, err := ioutil.ReadFile("test_exposecontroller_values.yaml")
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
	values.ExposeController.Config.HTTP = true
	values.ExposeController.Config.TLSAcme = false
	s, err := values.String()
	assert.NoError(t, err)
	assert.Equal(t, s, string(testFile), "expected exposecontroller helm values do not match")
}
