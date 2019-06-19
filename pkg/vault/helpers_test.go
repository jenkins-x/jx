package vault_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/jenkins-x/jx/pkg/vaulturl"
	"github.com/pborman/uuid"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/util"
	vault_test "github.com/jenkins-x/jx/pkg/vault/mocks"
	"github.com/petergtz/pegomock"
)

func TestReplaceURIs(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	vaultClient := vault_test.NewMockClient()
	path := "/baz/qux"
	key := "cheese"
	secret := uuid.New()
	valuesyaml := fmt.Sprintf(`foo:
  bar: vault:%s:%s
`, path, key)
	valuesFile, err := ioutil.TempFile("", "values.yaml")
	defer func() {
		err := util.DeleteFile(valuesFile.Name())
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	err = ioutil.WriteFile(valuesFile.Name(), []byte(valuesyaml), 0600)
	assert.NoError(t, err)
	pegomock.When(vaultClient.Read(pegomock.EqString(path))).ThenReturn(map[string]interface{}{
		key: secret,
	}, nil)
	result, err := vaulturl.ReplaceURIs(valuesyaml, vaultClient)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`foo:
  bar: %s
`, secret), result)
}

func TestReplaceRealExampleURI(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	vaultClient := vault_test.NewMockClient()
	path := "secret/gitOps/jenkins-x/environment-tekton-mole-dev/connectors-github-config-clientid-secret"
	key := "token-passthrough"
	secret := uuid.New()
	valuesyaml := fmt.Sprintf(`foo:
  bar: vault:%s:%s
`, path, key)
	valuesFile, err := ioutil.TempFile("", "values.yaml")
	defer func() {
		err := util.DeleteFile(valuesFile.Name())
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	err = ioutil.WriteFile(valuesFile.Name(), []byte(valuesyaml), 0600)
	assert.NoError(t, err)
	pegomock.When(vaultClient.Read(pegomock.EqString(path))).ThenReturn(map[string]interface{}{
		key: secret,
	}, nil)
	result, err := vaulturl.ReplaceURIs(valuesyaml, vaultClient)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`foo:
  bar: %s
`, secret), result)
}
