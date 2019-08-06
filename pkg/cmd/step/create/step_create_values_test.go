package create

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/require"

	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"

	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	clients_test "github.com/jenkins-x/jx/pkg/cmd/clients/mocks"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	vault_test "github.com/jenkins-x/jx/pkg/vault/mocks"

	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/petergtz/pegomock"
)

var timeout = 5 * time.Second

func TestGeneratedSchema(t *testing.T) {
	type testCase struct {
		GitKind, GitServer, ExpectedTokenDescription, ExpectedPattern string
		ExpectedMin, ExpectedMax                                      int
	}

	testCases := []testCase{
		{
			GitKind:                  "github",
			GitServer:                "",
			ExpectedTokenDescription: "A token for the Git user that will perform git operations inside a pipeline. This includes environment repository creation, and so this token should have full repository permissions. To create a token go to https://github.com/settings/tokens/new?scopes=repo,read:user,read:org,user:email,write:repo_hook,delete_repo then enter a name, click Generate token, and copy and paste the token into this prompt.",
			ExpectedMin:              40,
			ExpectedMax:              40,
			ExpectedPattern:          "^[0-9a-f]{40}$",
		},
		{
			GitKind:                  "bitbucketserver",
			GitServer:                "https://bitbucket.myorg.com",
			ExpectedTokenDescription: "A token for the Git user that will perform git operations inside a pipeline. This includes environment repository creation, and so this token should have full repository permissions. To create a token go to https://bitbucket.myorg.com/plugins/servlet/access-tokens/manage then enter a name, click Generate token, and copy and paste the token into this prompt.",
			ExpectedMin:              8,
			ExpectedMax:              50,
		},
		{
			GitKind:                  "gitlab",
			GitServer:                "https://gitlab.myorg.com",
			ExpectedTokenDescription: "A token for the Git user that will perform git operations inside a pipeline. This includes environment repository creation, and so this token should have full repository permissions. To create a token go to https://gitlab.myorg.com/profile/personal_access_tokens then enter a name, click Generate token, and copy and paste the token into this prompt.",
			ExpectedMin:              8,
			ExpectedMax:              50,
		},
	}

	for _, tc := range testCases {
		sourceData := filepath.Join("test_data", "step_create_values", "install")
		assert.DirExists(t, sourceData)

		testData, err := ioutil.TempDir("", "test-jx-step-create-values-")
		assert.NoError(t, err)

		err = util.CopyDir(sourceData, testData, true)
		assert.NoError(t, err)
		assert.DirExists(t, testData)

		requirements, requirementsFile, err := config.LoadRequirementsConfig(testData)
		require.NoError(t, err, "failed to load requirements in dir %s", testData)

		requirements.Cluster.GitKind = tc.GitKind
		requirements.Cluster.GitServer = tc.GitServer
		err = requirements.SaveConfig(requirementsFile)
		require.NoError(t, err, "failed to save requirements as file %s", requirementsFile)

		mockFactory := clients_test.NewMockFactory()
		commonOpts := opts.NewCommonOptionsWithFactory(mockFactory)
		o := StepCreateValuesOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &commonOpts,
				},
			},
			Dir: testData,
		}

		schemaFile := filepath.Join(testData, "values.schema.json")
		err = o.templateSchemaFile(schemaFile, requirements)
		require.NoError(t, err, "failed to generate template schema file %s", schemaFile)
		assert.FileExists(t, schemaFile)

		data, err := ioutil.ReadFile(schemaFile)
		require.NoError(t, err, "failed to load schema file %s", schemaFile)

		m := map[string]interface{}{}
		err = json.Unmarshal(data, &m)
		require.NoError(t, err, "failed to parse schema file %s", schemaFile)

		actual := util.GetMapValueAsStringViaPath(m, "properties.pipelineUser.properties.token.description")
		t.Logf("git token for git kind %s and server %s has description %s\n", tc.GitKind, tc.GitServer, actual)
		assert.Equal(t, tc.ExpectedTokenDescription, actual, "properties.pipelineUser.properties.token.description in generated schema for git kind %s and server %s", tc.GitKind, tc.GitServer)

		pattern := util.GetMapValueAsStringViaPath(m, "properties.pipelineUser.properties.token.pattern")
		t.Logf("git token for git kind %s and server %s has pattern: %s\n", tc.GitKind, tc.GitServer, pattern)
		assert.Equal(t, tc.ExpectedPattern, pattern, "properties.pipelineUser.properties.token.pattern in generated schema for git kind %s and server %s", tc.GitKind, tc.GitServer)

		min := util.GetMapValueAsIntViaPath(m, "properties.pipelineUser.properties.token.minLength")
		t.Logf("git token for git kind %s and server %s has minLength %d\n", tc.GitKind, tc.GitServer, min)
		assert.Equal(t, tc.ExpectedMin, min, "properties.pipelineUser.properties.token.minLength in generated schema for git kind %s and server %s", tc.GitKind, tc.GitServer)

		max := util.GetMapValueAsIntViaPath(m, "properties.pipelineUser.properties.token.maxLength")
		t.Logf("git token for git kind %s and server %s has maxLength %d\n", tc.GitKind, tc.GitServer, max)
		assert.Equal(t, tc.ExpectedMax, max, "properties.pipelineUser.properties.token.maxLength in generated schema for git kind %s and server %s", tc.GitKind, tc.GitServer)
	}
}

func TestCreateValuesFileWithVault(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")

	sourceData := filepath.Join("test_data", "step_create_values", "install")
	assert.DirExists(t, sourceData)

	testData, err := ioutil.TempDir("", "test-jx-step-create-values-")
	assert.NoError(t, err)

	err = util.CopyDir(sourceData, testData, true)
	assert.NoError(t, err)
	assert.DirExists(t, testData)

	pegomock.RegisterMockTestingT(t)
	tests.Retry(t, 1, time.Second*10, func(r *tests.R) {
		testOrgNameUUID, err := uuid.NewV4()
		assert.NoError(t, err)
		testOrgName := testOrgNameUUID.String()
		testRepoNameUUID, err := uuid.NewV4()
		assert.NoError(t, err)
		testRepoName := testRepoNameUUID.String()
		devEnvRepoName := fmt.Sprintf("environment-%s-%s-dev", testOrgName, testRepoName)
		devEnvRepo, _ := gits.NewFakeRepository(testOrgName, devEnvRepoName, nil, nil)
		mockFactory := clients_test.NewMockFactory()
		commonOpts := opts.NewCommonOptionsWithFactory(mockFactory)
		mockVaultClient := vault_test.NewMockClient()
		devEnv := kube.NewPermanentEnvironmentWithGit("dev", fmt.Sprintf("https://fake.git/%s/%s.git", testOrgName,
			devEnvRepoName))
		devEnv.Spec.Source.URL = devEnvRepo.GitRepo.CloneURL
		devEnv.Spec.Source.Ref = "master"
		pegomock.When(mockFactory.SecretsLocation()).ThenReturn(pegomock.ReturnValue(secrets.VaultLocationKind))
		pegomock.When(mockFactory.CreateSystemVaultClient(pegomock.AnyString())).ThenReturn(pegomock.ReturnValue(mockVaultClient), pegomock.ReturnValue(nil))
		mockHelmer := helm_test.NewMockHelmer()
		installerMock := resources_test.NewMockInstaller()
		testhelpers.ConfigureTestOptionsWithResources(&commonOpts,
			[]runtime.Object{},
			[]runtime.Object{
				devEnv,
			},
			gits.NewGitLocal(),
			nil,
			mockHelmer,
			installerMock,
		)
		testhelpers.MockFactoryWithKubeClients(mockFactory, &commonOpts)

		console := tests.NewTerminal(r, &timeout)
		defer console.Cleanup()
		commonOpts.In = console.In
		commonOpts.Out = console.Out
		commonOpts.Err = console.Err

		commonOpts.BatchMode = false

		outFile, err := ioutil.TempFile("", "")
		assert.NoError(t, err)

		o := StepCreateValuesOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &commonOpts,
				},
			},
			Dir:           testData,
			Name:          "values",
			SecretsScheme: "vault",
			ValuesFile:    outFile.Name(),
		}

		donec := make(chan struct{})
		go func() {
			defer close(donec)
			console.ExpectString("Jenkins X Admin Username")
			console.SendLine("admin")
			console.ExpectString("Jenkins X Admin Password")
			console.SendLine("abc")
			console.ExpectString("HMAC token")
			console.SendLine("abc")
			console.ExpectString("Pipeline bot Git username")
			console.SendLine("james")
			console.ExpectString("Pipeline bot Git token")
			console.SendLine("123456789")
			console.ExpectString("Do you want to configure a Docker Registry?")
			console.SendLine("y")
			console.ExpectString("Docker Registry URL")
			console.SendLine("")
			console.ExpectString("Docker Registry username")
			console.SendLine("james")
			console.ExpectString("Docker Registry password")
			console.SendLine("abc")
			console.ExpectString("Do you want to configure a GPG Key?")
			console.SendLine("n")
			console.ExpectEOF()
		}()
		err = o.Run()
		assert.NoError(r, err)
		console.Close()
		<-donec
		r.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

		// template the goldenfile to insert the generated org and repo name
		values := map[string]string{
			"org":  testOrgName,
			"repo": testRepoName,
		}
		goldenTmplBytes, err := ioutil.ReadFile(filepath.Join(testData, "values.yaml.golden"))
		assert.NoError(t, err)
		goldenTmplStr := string(goldenTmplBytes)
		goldenTmpl, err := template.New("goldenBytes").Parse(goldenTmplStr)
		assert.NoError(t, err)
		var goldenBytes bytes.Buffer
		err = goldenTmpl.Execute(&goldenBytes, values)
		assert.NoError(t, err)

		actual, err := ioutil.ReadFile(outFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, goldenBytes.String(), string(actual))
	})
}
