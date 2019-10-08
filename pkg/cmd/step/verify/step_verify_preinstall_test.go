package verify

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/acarl005/stripansi"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

var timeout = 1 * time.Second

func Test_verifyPrivateRepos_returns_nil_in_batch_mode(t *testing.T) {
	t.Parallel()
	log.SetOutput(ioutil.Discard)

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					BatchMode: true,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}

	assert.NoError(t, testOptions.verifyPrivateRepos(testConfig))
}

func Test_confirm_private_repos_with_github_provider(t *testing.T) {
	t.Parallel()
	log.SetOutput(ioutil.Discard)

	console := tests.NewTerminal(t, &timeout)
	defer console.Cleanup()

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					In:  console.In,
					Out: console.Out,
					Err: console.Err,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}
	testConfig.Cluster.GitKind = "github"
	testConfig.Cluster.EnvironmentGitOwner = "acme"

	done := make(chan struct{})
	go func() {
		defer close(done)
		console.ExpectString("If 'acme' is an GitHub organisation it needs to have a paid subscription to create private repos. Do you wish to continue?")
		console.SendLine("Y")
		console.ExpectEOF()
	}()
	err := testOptions.verifyPrivateRepos(testConfig)
	console.Close()
	<-done

	assert.NoError(t, err)
}

func Test_doesnt_ask_for_confirmation_when_in_gke(t *testing.T) {
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	testOptions := &StepVerifyPreInstallOptions{
		WorkloadIdentity: true,
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					Out: fakeStdout,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}
	testConfig.Cluster.GitKind = "github"
	testConfig.Cluster.Provider = "gke"
	testConfig.Cluster.EnvironmentGitOwner = "acme"
	testConfig.Cluster.ProjectID = "test"
	testConfig.Cluster.Zone = "exzone"
	testConfig.Cluster.ClusterName = "acme"

	testOptions.gatherRequirements(testConfig, "")
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.NotContains(t, output, fmt.Sprintf("jx boot has only been validated on GKE"))
}

func Test_doesnt_ask_for_confirmation_when_in_batch_mode_and_with_different_provider(t *testing.T) {
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					BatchMode: true,
					Out:       fakeStdout,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}
	testConfig.Cluster.GitKind = "github"
	testConfig.Cluster.Provider = "iks"
	testConfig.Cluster.EnvironmentGitOwner = "acme"
	testConfig.Cluster.ProjectID = "test"
	testConfig.Cluster.Zone = "exzone"
	testConfig.Cluster.ClusterName = "acme"

	testOptions.gatherRequirements(testConfig, "")
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, fmt.Sprintf("jx boot has only been validated on GKE"))
}

func Test_asks_for_confirmation_when_not_in_batch_mode_and_with_different_provider(t *testing.T) {
	t.Parallel()
	log.SetOutput(ioutil.Discard)

	console := tests.NewTerminal(t, &timeout)
	defer console.Cleanup()

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					BatchMode: false,
					In:        console.In,
					Out:       console.Out,
					Err:       console.Err,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}
	testConfig.Cluster.GitKind = "github"
	testConfig.Cluster.Provider = "iks"
	testConfig.Cluster.EnvironmentGitOwner = "acme"
	testConfig.Cluster.ProjectID = "test"
	testConfig.Cluster.Zone = "exzone"
	testConfig.Cluster.ClusterName = "acme"

	done := make(chan struct{})
	go func() {
		defer close(done)
		console.ExpectString("Continue execution anyway?")
		console.SendLine("N")
		console.ExpectEOF()
	}()
	testOptions.gatherRequirements(testConfig, "")
	console.Close()
	<-done
}

func Test_abort_private_repos_with_github_provider(t *testing.T) {
	t.Parallel()
	log.SetOutput(ioutil.Discard)

	console := tests.NewTerminal(t, &timeout)
	defer console.Cleanup()

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: &opts.CommonOptions{
					In:  console.In,
					Out: console.Out,
					Err: console.Err,
				},
			},
		},
	}

	testConfig := &config.RequirementsConfig{}
	testConfig.Cluster.GitKind = "github"
	testConfig.Cluster.EnvironmentGitOwner = "acme"

	done := make(chan struct{})
	go func() {
		defer close(done)
		console.ExpectString("If 'acme' is an GitHub organisation it needs to have a paid subscription to create private repos. Do you wish to continue?")
		console.SendLine("N")
		console.ExpectEOF()
	}()
	err := testOptions.verifyPrivateRepos(testConfig)
	console.Close()
	<-done

	assert.Error(t, err)
	assert.Equal(t, "cannot continue without completed git requirements", err.Error())
}

func TestStepVerifyPreInstallOptions_VerifyRequirementsInTeamSettings(t *testing.T) {
	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	testOptions := &StepVerifyPreInstallOptions{
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

	err = testOptions.VerifyRequirementsInTeamSettings("jx", requirements)
	assert.NoError(t, err, "there shouldn't be any error adding the requirements to TeamSettings")

	teamSettings, err := testOptions.TeamSettings()
	assert.NoError(t, err)

	requirementsCm := teamSettings.BootRequirements
	assert.NotEqual(t, "", requirementsCm, "the BootRequirements field should be present and not empty")

	mapRequirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
	assert.NoError(t, err)

	assert.Equal(t, requirements, mapRequirements)
}

func TestStepVerifyPreInstallOptions_VerifyRequirementsConfigMapWithModification(t *testing.T) {
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

	testOptions := &StepVerifyPreInstallOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: options,
			},
		},
	}

	err = testOptions.VerifyRequirementsInTeamSettings("jx", requirements)
	assert.NoError(t, err, "there shouldn't be any error creating the ConfigMap")

	teamSettings, err := testOptions.TeamSettings()
	assert.NoError(t, err)

	requirementsCm := teamSettings.BootRequirements
	assert.NotEqual(t, "", requirementsCm, "the BootRequirements field should be present and not empty")

	mapRequirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
	assert.NoError(t, err)

	assert.Equal(t, requirements.Storage.Logs, mapRequirements.Storage.Logs, "the change done before calling"+
		"VerifyRequirementsInTeamSettings should be present in the retrieved configuration")
}
