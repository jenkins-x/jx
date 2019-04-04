package testhelpers

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler/prow"

	"k8s.io/test-infra/prow/plugins"

	"k8s.io/test-infra/prow/config"

	"github.com/stretchr/testify/assert"

	"github.com/pborman/uuid"
)

// CompleteScheduler returns a SchedulerSpec completely filled with dummy data
func CompleteScheduler() *v1.SchedulerSpec {
	return &v1.SchedulerSpec{
		Policy: pointerToGlobalProtectionPolicy(),
		Merger: &v1.Merger{
			ContextPolicy: &v1.ContextPolicy{
				OptionalContexts:          pointerToReplaceableSliceOfStrings(),
				RequiredContexts:          pointerToReplaceableSliceOfStrings(),
				RequiredIfPresentContexts: pointerToReplaceableSliceOfStrings(),
			},
			MergeType:          pointerToUUID(),
			TargetURL:          pointerToUUID(),
			PRStatusBaseURL:    pointerToUUID(),
			BlockerLabel:       pointerToUUID(),
			SquashLabel:        pointerToUUID(),
			MaxGoroutines:      pointerToRandomNumber(),
			StatusUpdatePeriod: pointerToRandomDuration(),
			SyncPeriod:         pointerToRandomDuration(),
		},
		Presubmits: &v1.Presubmits{
			Items: []*v1.Presubmit{
				{
					MergeType: pointerToUUID(),
					Context:   pointerToUUID(),
					Report:    pointerToTrue(),
					AlwaysRun: pointerToTrue(),
					Optional:  pointerToTrue(),
					ContextPolicy: &v1.RepoContextPolicy{
						ContextPolicy: pointerToContextPolicy(),
						Branches: &v1.ReplaceableMapOfStringContextPolicy{
							Items: map[string]*v1.ContextPolicy{
								uuid.New(): pointerToContextPolicy(),
							},
						},
					},
					Query: &v1.Query{
						Labels:                 pointerToReplaceableSliceOfStrings(),
						ExcludedBranches:       pointerToReplaceableSliceOfStrings(),
						IncludedBranches:       pointerToReplaceableSliceOfStrings(),
						MissingLabels:          pointerToReplaceableSliceOfStrings(),
						Milestone:              pointerToUUID(),
						ReviewApprovedRequired: pointerToTrue(),
					},
					Brancher:     pointerToBrancher(),
					RerunCommand: pointerToUUID(),
					Trigger:      pointerToUUID(),
					Policy: &v1.ProtectionPolicies{
						Items: map[string]*v1.ProtectionPolicy{
							uuid.New(): pointerToProtectionPolicy(),
						},
					},
					RegexpChangeMatcher: pointerToRegexpChangeMatcher(),
					JobBase:             pointerToJobBase(),
				},
			},
		},
		Postsubmits: &v1.Postsubmits{
			Items: []*v1.Postsubmit{
				{
					Report:              pointerToTrue(),
					Context:             pointerToUUID(),
					JobBase:             pointerToJobBase(),
					RegexpChangeMatcher: pointerToRegexpChangeMatcher(),
					Brancher:            pointerToBrancher(),
				},
			},
		},
		Trigger: &v1.Trigger{
			IgnoreOkToTest: pointerToTrue(),
			JoinOrgURL:     pointerToUUID(),
			OnlyOrgMembers: pointerToTrue(),
			TrustedOrg:     pointerToUUID(),
		},
		ScehdulerAgent: &v1.SchedulerAgent{
			Agent: pointerToUUID(),
		},
		Approve: &v1.Approve{
			RequireSelfApproval: pointerToTrue(),
			LgtmActsAsApprove:   pointerToTrue(),
			IssueRequired:       pointerToTrue(),
			IgnoreReviewState:   pointerToTrue(),
		},
		ExternalPlugins: &v1.ReplaceableSliceOfExternalPlugins{
			Items: []*v1.ExternalPlugin{
				{
					Name:     pointerToUUID(),
					Events:   pointerToReplaceableSliceOfStrings(),
					Endpoint: pointerToUUID(),
				},
			},
		},
		LGTM: &v1.Lgtm{
			StoreTreeHash:    pointerToTrue(),
			ReviewActsAsLgtm: pointerToTrue(),
			StickyLgtmTeam:   pointerToUUID(),
		},
		Plugins: pointerToReplaceableSliceOfStrings(),
	}
}

func pointerToTrue() *bool {
	b := true
	return &b
}

func pointerToUUID() *string {
	s := uuid.New()
	return &s
}

func pointerToRandomNumber() *int {
	i := rand.Int()
	return &i
}

func pointerToRandomDuration() *time.Duration {
	i := rand.Int63()
	duration := time.Duration(i)
	return &duration
}

func pointerToReplaceableSliceOfStrings() *v1.ReplaceableSliceOfStrings {
	return &v1.ReplaceableSliceOfStrings{
		Items: []string{
			uuid.New(),
		},
	}
}

func pointerToReplaceableMapOfStringString() *v1.ReplaceableMapOfStringString {
	return &v1.ReplaceableMapOfStringString{
		Items: map[string]string{
			uuid.New(): uuid.New(),
		},
	}
}

func pointerToContextPolicy() *v1.ContextPolicy {
	return &v1.ContextPolicy{
		SkipUnknownContexts:       pointerToTrue(),
		FromBranchProtection:      pointerToTrue(),
		RequiredIfPresentContexts: pointerToReplaceableSliceOfStrings(),
		RequiredContexts:          pointerToReplaceableSliceOfStrings(),
		OptionalContexts:          pointerToReplaceableSliceOfStrings(),
	}
}

func pointerToGlobalProtectionPolicy() *v1.GlobalProtectionPolicy {
	return &v1.GlobalProtectionPolicy{
		ProtectTested:    pointerToTrue(),
		ProtectionPolicy: pointerToProtectionPolicy(),
	}
}

func pointerToProtectionPolicy() *v1.ProtectionPolicy {
	return &v1.ProtectionPolicy{
		Restrictions: &v1.Restrictions{
			Users: pointerToReplaceableSliceOfStrings(),
			Teams: pointerToReplaceableSliceOfStrings(),
		},
		Admins: pointerToTrue(),
		RequiredPullRequestReviews: &v1.ReviewPolicy{
			DismissalRestrictions: &v1.Restrictions{
				Users: pointerToReplaceableSliceOfStrings(),
				Teams: pointerToReplaceableSliceOfStrings(),
			},
		},
		RequiredStatusChecks: &v1.BranchProtectionContextPolicy{
			Strict:   pointerToTrue(),
			Contexts: pointerToReplaceableSliceOfStrings(),
		},
		Protect: pointerToTrue(),
	}
}

func pointerToJobBase() *v1.JobBase {
	return &v1.JobBase{
		Labels:         pointerToReplaceableMapOfStringString(),
		Namespace:      pointerToUUID(),
		Cluster:        pointerToUUID(),
		MaxConcurrency: pointerToRandomNumber(),
		Agent:          pointerToUUID(),
		Name:           pointerToUUID(),
	}
}

func pointerToRegexpChangeMatcher() *v1.RegexpChangeMatcher {
	return &v1.RegexpChangeMatcher{
		RunIfChanged: pointerToUUID(),
	}
}

func pointerToBrancher() *v1.Brancher {
	return &v1.Brancher{
		Branches:     pointerToReplaceableSliceOfStrings(),
		SkipBranches: pointerToReplaceableSliceOfStrings(),
	}
}

// SchedulerFile contains a list of leaf files to build the scheduler from
type SchedulerFile struct {
	// Filenames is the hierarchy with the leaf at the right
	Filenames []string
	Org       string
	Repo      string
}

// BuildAndValidateProwConfig takes a list of schedulerFiles and builds them to a Prow config,
// and validates them against the expectedConfigFilename and expectedPluginsFilename that make up the prow config.
// Filepaths are relative to the baseDir
func BuildAndValidateProwConfig(t *testing.T, baseDir string, expectedConfigFilename string,
	expectedPluginsFilename string, schedulerFiles []SchedulerFile) {
	var expectedConfig config.Config
	if expectedConfigFilename != "" {
		cfgBytes, err := ioutil.ReadFile(filepath.Join(baseDir, expectedConfigFilename))
		assert.NoError(t, err)
		err = yaml.Unmarshal(cfgBytes, &expectedConfig)
		assert.NoError(t, err)
	}

	var expectedPlugins plugins.Configuration
	if expectedPluginsFilename != "" {
		bytes, err := ioutil.ReadFile(filepath.Join(baseDir, expectedPluginsFilename))
		assert.NoError(t, err)
		err = yaml.Unmarshal(bytes, &expectedPlugins)
		assert.NoError(t, err)
	}

	schedulerLeaves := make([]*pipelinescheduler.SchedulerLeaf, 0)
	for _, sfs := range schedulerFiles {
		schedulers := make([]*v1.SchedulerSpec, 0)
		for _, f := range sfs.Filenames {
			bytes, err := ioutil.ReadFile(filepath.Join(baseDir, f))
			assert.NoError(t, err)
			s := v1.SchedulerSpec{}
			err = yaml.Unmarshal(bytes, &s)
			assert.NoError(t, err)
			schedulers = append(schedulers, &s)
		}
		s, err := pipelinescheduler.Build(schedulers)
		assert.NoError(t, err)
		schedulerLeaves = append(schedulerLeaves, &pipelinescheduler.SchedulerLeaf{
			Repo:          sfs.Repo,
			Org:           sfs.Org,
			SchedulerSpec: s,
		})
	}

	cfg, plugs, err := prow.Build(schedulerLeaves)
	assert.NoError(t, err)
	if expectedConfigFilename != "" {
		assert.Equal(t, &expectedConfig, cfg)
	}
	if expectedPluginsFilename != "" {
		assert.Equal(t, &expectedPlugins, plugs)
	}
}
