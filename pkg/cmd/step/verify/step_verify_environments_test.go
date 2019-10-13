package verify

import (
	"io/ioutil"
	"path"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/util"

	"sigs.k8s.io/yaml"

	"fmt"
	"os"

	"github.com/stretchr/testify/assert"
)

func TestStepVerifyEnvironmentsOptions_StoreRequirementsInTeamSettings(t *testing.T) {
	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	testOptions := &StepVerifyEnvironmentsOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: options,
			},
		},
	}

	requirementsYamlFile := path.Join("test_data", "preinstall", "no_tls", "jx-requirements.yml")
	exists, err := util.FileExists(requirementsYamlFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	bytes, err := ioutil.ReadFile(requirementsYamlFile)
	assert.NoError(t, err)
	requirements := &config.RequirementsConfig{}
	err = yaml.Unmarshal(bytes, requirements)
	assert.NoError(t, err)

	err = testOptions.storeRequirementsInTeamSettings(requirements)
	assert.NoError(t, err, "there shouldn't be any error adding the requirements to TeamSettings")

	teamSettings, err := testOptions.TeamSettings()
	assert.NoError(t, err)

	requirementsCm := teamSettings.BootRequirements
	assert.NotEqual(t, "", requirementsCm, "the BootRequirements field should be present and not empty")

	mapRequirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
	assert.NoError(t, err)

	assert.Equal(t, requirements, mapRequirements)
}

func TestStepVerifyEnvironmentsOptions_StoreRequirementsConfigMapWithModification(t *testing.T) {
	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	requirementsYamlFile := path.Join("test_data", "preinstall", "no_tls", "jx-requirements.yml")
	exists, err := util.FileExists(requirementsYamlFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	bytes, err := ioutil.ReadFile(requirementsYamlFile)
	assert.NoError(t, err)
	requirements := &config.RequirementsConfig{}
	err = yaml.Unmarshal(bytes, requirements)
	assert.NoError(t, err)

	err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
		env.Spec.TeamSettings.BootRequirements = string(bytes)
		return nil
	})
	assert.NoError(t, err)

	// We make a modification to the requirements and we should see it when we retrieve the ConfigMap later
	requirements.Storage.Logs = config.StorageEntryConfig{
		Enabled: true,
		URL:     "gs://randombucket",
	}

	testOptions := &StepVerifyEnvironmentsOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: options,
			},
		},
	}

	err = testOptions.storeRequirementsInTeamSettings(requirements)
	assert.NoError(t, err, "there shouldn't be any error updating the team settings")

	teamSettings, err := testOptions.TeamSettings()
	assert.NoError(t, err)

	requirementsCm := teamSettings.BootRequirements
	assert.NotEqual(t, "", requirementsCm, "the BootRequirements field should be present and not empty")

	mapRequirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
	assert.NoError(t, err)

	assert.Equal(t, requirements.Storage.Logs, mapRequirements.Storage.Logs, "the change done before calling"+
		"VerifyRequirementsInTeamSettings should be present in the retrieved configuration")
}

func Test_IsJXBoot(t *testing.T) {
	orig, found := os.LookupEnv(jxInterpretPipelineEnvKey)
	defer func() {
		if found {
			_ = os.Setenv(jxInterpretPipelineEnvKey, orig)
		}
	}()

	o := StepVerifyEnvironmentsOptions{}

	err := os.Setenv(jxInterpretPipelineEnvKey, "")
	assert.NoError(t, err)
	assert.False(t, o.isJXBoot(), "we should not be in boot mode")

	err = os.Setenv(jxInterpretPipelineEnvKey, "false")
	assert.NoError(t, err)
	assert.False(t, o.isJXBoot(), "we should not be in boot mode")

	err = os.Setenv(jxInterpretPipelineEnvKey, "true")
	assert.NoError(t, err)
	assert.True(t, o.isJXBoot(), "we should be in boot mode")
}

func Test_ReadEnvironment(t *testing.T) {
	origConfigRepoURL, foundConfigRepoURLEnvKey := os.LookupEnv(configRepoURLEnvKey)
	origConfigRepoRef, foundConfigRepoRefEnvKey := os.LookupEnv(configRepoRefEnvKey)
	defer func() {
		if foundConfigRepoURLEnvKey {
			_ = os.Setenv(configRepoURLEnvKey, origConfigRepoURL)
		}

		if foundConfigRepoRefEnvKey {
			_ = os.Setenv(configRepoRefEnvKey, origConfigRepoRef)
		}
	}()

	o := StepVerifyEnvironmentsOptions{}

	var tests = []struct {
		url         string
		ref         string
		expectError bool
		errorString string
	}{
		{"https://github.com/jenkins-x/jenkins-x-boot-config", "master", false, ""},
		{"https://github.com/jenkins-x/jenkins-x-boot-config", "", true, "the environment variable BASE_CONFIG_REF must be specified"},
		{"", "master", true, "the environment variable REPO_URL must be specified"},
		{"", "", true, "[the environment variable REPO_URL must be specified, the environment variable BASE_CONFIG_REF must be specified]"},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprintf("%s-%s", testCase.url, testCase.ref), func(t *testing.T) {
			if testCase.url == "" {
				err := os.Unsetenv(configRepoURLEnvKey)
				assert.NoError(t, err)
			} else {
				err := os.Setenv(configRepoURLEnvKey, testCase.url)
				assert.NoError(t, err)
			}

			if testCase.ref == "" {
				err := os.Unsetenv(configRepoRefEnvKey)
				assert.NoError(t, err)
			} else {
				err := os.Setenv(configRepoRefEnvKey, testCase.ref)
				assert.NoError(t, err)

			}

			repo, ref, err := o.readEnvironment()
			if testCase.expectError {
				assert.Error(t, err)
				assert.Equal(t, testCase.errorString, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.url, repo)
				assert.Equal(t, testCase.ref, ref)
			}
		})
	}
}
