// +build unit

package promote_test

import (
	"sort"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/tests"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/promote"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeSearchForChart(f string) (string, error) {
	return "mySearchedApp", nil
}

func fakeDiscoverAppName() (string, error) {
	return "myDiscoveredApp", nil
}

func TestEnsureApplicationNameIsDefinedWithoutApplicationFlagWithArgs(t *testing.T) {
	promoteOptions := &promote.PromoteOptions{
		Environment: "production", // --env production
	}

	commonOpts := &opts.CommonOptions{}
	promoteOptions.CommonOptions = commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	commonOpts.Args = []string{"myArgumentApp"}

	err := promoteOptions.EnsureApplicationNameIsDefined(fakeSearchForChart, fakeDiscoverAppName)
	assert.NoError(t, err)

	assert.Equal(t, "myArgumentApp", promoteOptions.Application)
}

func TestEnsureApplicationNameIsDefinedWithoutApplicationFlagWithFilterFlag(t *testing.T) {
	promoteOptions := &promote.PromoteOptions{
		Environment: "production", // --env production
		Filter:      "something",
	}

	commonOpts := &opts.CommonOptions{}
	promoteOptions.CommonOptions = commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources

	err := promoteOptions.EnsureApplicationNameIsDefined(fakeSearchForChart, fakeDiscoverAppName)
	assert.NoError(t, err)

	assert.Equal(t, "mySearchedApp", promoteOptions.Application)
}

func TestEnsureApplicationNameIsDefinedWithoutApplicationFlagWithBatchFlag(t *testing.T) {
	promoteOptions := &promote.PromoteOptions{
		Environment: "production", // --env production
	}

	commonOpts := &opts.CommonOptions{}
	promoteOptions.CommonOptions = commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true           // --batch-mode

	err := promoteOptions.EnsureApplicationNameIsDefined(fakeSearchForChart, fakeDiscoverAppName)
	assert.NoError(t, err)

	assert.Equal(t, "myDiscoveredApp", promoteOptions.Application)
}

func TestEnsureApplicationNameIsDefinedWithoutApplicationFlag(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")

	console := tests.NewTerminal(t, nil)
	defer console.Cleanup()

	promoteOptions := &promote.PromoteOptions{
		Environment: "production", // --env production
	}

	commonOpts := &opts.CommonOptions{}
	promoteOptions.CommonOptions = commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.Out = console.Out
	promoteOptions.In = console.In

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		// Test boolean type
		console.ExpectString("Are you sure you want to promote the application named: myDiscoveredApp?")
		console.SendLine("Y")
		console.ExpectEOF()
	}()

	err := promoteOptions.EnsureApplicationNameIsDefined(fakeSearchForChart, fakeDiscoverAppName)

	console.Close()
	<-donec

	assert.NoError(t, err)
	assert.Equal(t, "myDiscoveredApp", promoteOptions.Application)
}

func TestEnsureApplicationNameIsDefinedWithoutApplicationFlagUserSaysNo(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")

	console := tests.NewTerminal(t, nil)
	defer console.Cleanup()

	promoteOptions := &promote.PromoteOptions{
		Environment: "production", // --env production
	}

	commonOpts := &opts.CommonOptions{}
	promoteOptions.CommonOptions = commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.Out = console.Out
	promoteOptions.In = console.In

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		// Test boolean type
		console.ExpectString("Are you sure you want to promote the application named: myDiscoveredApp?")
		console.SendLine("N")
		console.ExpectEOF()
	}()

	err := promoteOptions.EnsureApplicationNameIsDefined(fakeSearchForChart, fakeDiscoverAppName)

	console.Close()
	<-donec

	assert.Error(t, err)
	assert.Equal(t, "", promoteOptions.Application)
}

func TestGetEnvChartValues(t *testing.T) {
	tests := []struct {
		ns           string
		env          v1.Environment
		values       []string
		valueStrings []string
	}{{
		"jx-test-preview-pr-6",
		v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-preview",
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         "jx-test-preview-pr-6",
				Label:             "Test preview",
				Kind:              v1.EnvironmentKindTypePreview,
				PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
				PullRequestURL:    "https://github.com/my-project/my-app/pull/6",
				Order:             999,
				Source: v1.EnvironmentRepository{
					Kind: v1.EnvironmentRepositoryTypeGit,
					URL:  "https://github.com/my-project/my-app",
					Ref:  "my-branch",
				},
				PreviewGitSpec: v1.PreviewGitSpec{
					ApplicationName: "my-app",
					Name:            "6",
					URL:             "https://github.com/my-project/my-app/pull/6",
				},
			},
		},
		[]string{
			"tags.jx-preview=true",
			"tags.jx-env-test-preview=true",
			"tags.jx-ns-jx-test-preview-pr-6=true",
			"global.jxPreview=true",
			"global.jxEnvTestPreview=true",
			"global.jxNsJxTestPreviewPr6=true",
		},
		[]string{
			"global.jxTypeEnv=preview",
			"global.jxEnv=test-preview",
			"global.jxNs=jx-test-preview-pr-6",
			"global.jxPreviewApp=my-app",
			"global.jxPreviewPr=6",
		},
	}, {
		"jx-custom-env",
		v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "custom-env",
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         "jx-custom-env",
				Label:             "Custom environment",
				Kind:              v1.EnvironmentKindTypePermanent,
				PromotionStrategy: v1.PromotionStrategyTypeManual,
				Order:             5,
				Source: v1.EnvironmentRepository{
					Kind: v1.EnvironmentRepositoryTypeGit,
					URL:  "https://github.com/my-project/jx-environment-custom-env",
					Ref:  "master",
				},
			},
		},
		[]string{
			"tags.jx-permanent=true",
			"tags.jx-env-custom-env=true",
			"tags.jx-ns-jx-custom-env=true",
			"global.jxPermanent=true",
			"global.jxEnvCustomEnv=true",
			"global.jxNsJxCustomEnv=true",
		},
		[]string{
			"global.jxTypeEnv=permanent",
			"global.jxEnv=custom-env",
			"global.jxNs=jx-custom-env",
		},
	}, {
		"ns-rand",
		v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-env",
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         "ns-other",
				Label:             "Random environment",
				Kind:              v1.EnvironmentKindTypeEdit,
				PromotionStrategy: v1.PromotionStrategyTypeNever,
				Order:             666,
				Source: v1.EnvironmentRepository{
					Kind: v1.EnvironmentRepositoryTypeGit,
					URL:  "https://github.com/my-project/random",
					Ref:  "master",
				},
				PreviewGitSpec: v1.PreviewGitSpec{
					ApplicationName: "random",
					Name:            "2",
					URL:             "https://github.com/my-project/random/pull/6",
				},
			},
		},
		[]string{
			"tags.jx-edit=true",
			"tags.jx-env-random-env=true",
			"tags.jx-ns-ns-rand=true",
			"global.jxEdit=true",
			"global.jxEnvRandomEnv=true",
			"global.jxNsNsRand=true",
		},
		[]string{
			"global.jxTypeEnv=edit",
			"global.jxEnv=random-env",
			"global.jxNs=ns-rand",
		},
	}}

	for _, test := range tests {
		promoteOptions := &promote.PromoteOptions{}
		values, valueStrings := promoteOptions.GetEnvChartValues(test.ns, &test.env)
		sort.Strings(values)
		sort.Strings(test.values)
		assert.Equal(t, values, test.values)
		sort.Strings(valueStrings)
		sort.Strings(test.valueStrings)
		assert.Equal(t, valueStrings, test.valueStrings)
	}
}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type TestEnv struct {
	Activity        *v1.PipelineActivity
	CommonOptions   *opts.CommonOptions
	FakeGitProvider *gits.FakeProvider
	DevRepo         *gits.FakeRepository
	StagingRepo     *gits.FakeRepository
	ProdRepo        *gits.FakeRepository
}
