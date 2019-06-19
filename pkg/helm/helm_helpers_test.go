package helm_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/secreturl/localvault"
	"github.com/pborman/uuid"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/require"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"
	vault_test "github.com/jenkins-x/jx/pkg/vault/mocks"
	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
)

func TestCombineMapTrees(t *testing.T) {
	t.Parallel()

	assertCombineMapTrees(t,
		map[string]interface{}{
			"foo": "foovalue",
			"bar": "barvalue",
		},
		map[string]interface{}{
			"foo": "foovalue",
		},
		map[string]interface{}{
			"bar": "barvalue",
		},
	)

	assertCombineMapTrees(t,
		map[string]interface{}{
			"child": map[string]interface{}{
				"foo": "foovalue",
				"bar": "barvalue",
			},
			"m1": map[string]interface{}{
				"thingy": "thingyvalue",
			},
			"m2": map[string]interface{}{
				"another": "anothervalue",
			},
		},
		map[string]interface{}{
			"child": map[string]interface{}{
				"foo": "foovalue",
			},
			"m1": map[string]interface{}{
				"thingy": "thingyvalue",
			},
		},
		map[string]interface{}{
			"child": map[string]interface{}{
				"bar": "barvalue",
			},
			"m2": map[string]interface{}{
				"another": "anothervalue",
			},
		},
	)
}

func assertCombineMapTrees(t *testing.T, expected map[string]interface{}, destination map[string]interface{}, input map[string]interface{}) {
	actual := map[string]interface{}{}
	for k, v := range destination {
		actual[k] = v
	}

	util.CombineMapTrees(actual, input)

	assert.Equal(t, actual, expected, "when combine map trees", mapToString(destination), mapToString(input))
}

func mapToString(m map[string]interface{}) string {
	return fmt.Sprintf("%#v", m)
}

func TestSetValuesToMap(t *testing.T) {
	t.Parallel()

	setValues := []string{"foo.bar=abc", "cheese=def"}
	actual := helm.SetValuesToMap(setValues)

	expected := map[string]interface{}{
		"cheese": "def",
		"foo": map[string]interface{}{
			"bar": "abc",
		},
	}
	assert.Equal(t, actual, expected, "setValuesToMap for values %s", strings.Join(setValues, ", "))
}

func TestStoreCredentials(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	vaultClient := vault_test.NewMockClient()
	repository := "http://charts.acme.com"
	username := uuid.New()
	password := uuid.New()
	username, password, err := helm.DecorateWithCredentials(repository, username, password, vaultClient, nil, nil, nil)
	assert2.NoError(t, err)
	vaultClient.VerifyWasCalledOnce().WriteObject(helm.RepoVaultPath, helm.HelmRepoCredentials{
		repository: helm.HelmRepoCredential{
			Username: username,
			Password: password,
		},
	})
}

func TestRetrieveCredentials(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	vaultClient := vault_test.NewMockClient()
	repository := "http://charts.acme.com"
	username := uuid.New()
	password := uuid.New()
	pegomock.When(vaultClient.ReadObject(pegomock.EqString(helm.RepoVaultPath),
		pegomock.AnyInterface())).Then(func(params []pegomock.Param) pegomock.ReturnValues {
		p := params[1].(*helm.HelmRepoCredentials)
		secrets := *p
		secret := helm.HelmRepoCredential{
			Username: username,
			Password: password,
		}
		secrets[repository] = secret
		return []pegomock.ReturnValue{
			nil,
		}
	})
	retrievedUsername, retrievedPassword, err := helm.DecorateWithCredentials(repository, "", "", vaultClient, nil,
		nil, nil)
	assert2.NoError(t, err)
	vaultClient.VerifyWasCalledOnce().ReadObject(pegomock.EqString(helm.RepoVaultPath), pegomock.AnyInterface())
	assert2.Equal(t, username, retrievedUsername)
	assert2.Equal(t, password, retrievedPassword)
}

func TestOverrideCredentials(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	vaultClient := vault_test.NewMockClient()
	repository := "http://charts.acme.com"
	username := uuid.New()
	password := uuid.New()
	newUsername := uuid.New()
	newPassword := uuid.New()
	pegomock.When(vaultClient.ReadObject(pegomock.EqString(helm.RepoVaultPath),
		pegomock.AnyInterface())).Then(func(params []pegomock.Param) pegomock.ReturnValues {
		p := params[1].(*helm.HelmRepoCredentials)
		secrets := *p
		secret := helm.HelmRepoCredential{
			Username: username,
			Password: password,
		}
		secrets[repository] = secret
		return []pegomock.ReturnValue{
			nil,
		}
	})
	retrievedUsername, retrievedPassword, err := helm.DecorateWithCredentials(repository, newUsername, newPassword,
		vaultClient, nil, nil, nil)
	assert2.NoError(t, err)
	assert2.Equal(t, newUsername, retrievedUsername)
	assert2.Equal(t, newPassword, retrievedPassword)
	vaultClient.VerifyWasCalledOnce().WriteObject(helm.RepoVaultPath, helm.HelmRepoCredentials{
		repository: helm.HelmRepoCredential{
			Username: newUsername,
			Password: newPassword,
		},
	})
}

func TestReplaceVaultURI(t *testing.T) {
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
		assert2.NoError(t, err)
	}()
	assert2.NoError(t, err)
	err = ioutil.WriteFile(valuesFile.Name(), []byte(valuesyaml), 0600)
	assert2.NoError(t, err)
	options := helm.InstallChartOptions{
		ValueFiles: []string{
			valuesFile.Name(),
		},
	}
	pegomock.When(vaultClient.Read(pegomock.EqString(path))).ThenReturn(map[string]interface{}{
		key: secret,
	}, nil)
	cleanup, err := helm.DecorateWithSecrets(&options, vaultClient)
	defer cleanup()
	assert2.Len(t, options.ValueFiles, 1)
	newValuesYaml, err := ioutil.ReadFile(options.ValueFiles[0])
	assert2.NoError(t, err)
	assert2.Equal(t, fmt.Sprintf(`foo:
  bar: %s
`, secret), string(newValuesYaml))
}

func TestReplaceVaultURIWithLocalFile(t *testing.T) {
	vaultClient := localvault.NewFileSystemClient(path.Join("test_data", "local_vault_files"))
	path := "/baz/qux"
	key := "cheese"
	secret := "Edam"
	valuesyaml := fmt.Sprintf(`foo:
  bar: vault:%s:%s
`, path, key)
	valuesFile, err := ioutil.TempFile("", "values.yaml")
	defer func() {
		err := util.DeleteFile(valuesFile.Name())
		assert2.NoError(t, err)
	}()
	assert2.NoError(t, err)
	err = ioutil.WriteFile(valuesFile.Name(), []byte(valuesyaml), 0600)
	assert2.NoError(t, err)
	options := helm.InstallChartOptions{
		ValueFiles: []string{
			valuesFile.Name(),
		},
	}

	actual, err := vaultClient.Read(path)
	expected := map[string]interface{}{
		key: secret,
	}

	require.NoError(t, err, "reading vault client on path %s", path)
	assert2.Equal(t, expected, actual, "vault read at path %s", path)

	cleanup, err := helm.DecorateWithSecrets(&options, vaultClient)
	defer cleanup()
	assert2.Len(t, options.ValueFiles, 1)
	newValuesYaml, err := ioutil.ReadFile(options.ValueFiles[0])
	assert2.NoError(t, err)
	assert2.Equal(t, fmt.Sprintf(`foo:
  bar: %s
`, secret), string(newValuesYaml))
}
