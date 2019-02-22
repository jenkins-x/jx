package test_helpers

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler/prow"

	"k8s.io/test-infra/prow/plugins"

	"k8s.io/test-infra/prow/config"

	"github.com/stretchr/testify/assert"

	"github.com/pborman/uuid"
)

// CompleteScheduler returns a Scheduler completely filled with dummy data
func CompleteScheduler() *pipelinescheduler.Scheduler {
	return &pipelinescheduler.Scheduler{
		Policy: pointerToGlobalProtectionPolicy(),
		Merger: &pipelinescheduler.Merger{
			ContextPolicy: &pipelinescheduler.ContextPolicy{
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
		Presubmits: &pipelinescheduler.Presubmits{
			Items: []*pipelinescheduler.Presubmit{
				&pipelinescheduler.Presubmit{
					MergeType: pointerToUUID(),
					Context:   pointerToUUID(),
					Report:    pointerToTrue(),
					AlwaysRun: pointerToTrue(),
					Optional:  pointerToTrue(),
					ContextPolicy: &pipelinescheduler.RepoContextPolicy{
						ContextPolicy: pointerToContextPolicy(),
						Branches: &pipelinescheduler.ReplaceableMapOfStringContextPolicy{
							Items: map[string]*pipelinescheduler.ContextPolicy{
								uuid.New(): pointerToContextPolicy(),
							},
						},
					},
					Query: &pipelinescheduler.Query{
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
					Policy: &pipelinescheduler.ProtectionPolicies{
						Items: map[string]*pipelinescheduler.ProtectionPolicy{
							uuid.New(): pointerToProtectionPolicy(),
						},
					},
					RegexpChangeMatcher: pointerToRegexpChangeMatcher(),
					JobBase:             pointerToJobBase(),
				},
			},
		},
		Postsubmits: &pipelinescheduler.Postsubmits{
			Items: []*pipelinescheduler.Postsubmit{
				&pipelinescheduler.Postsubmit{
					Report:              pointerToTrue(),
					Context:             pointerToUUID(),
					JobBase:             pointerToJobBase(),
					RegexpChangeMatcher: pointerToRegexpChangeMatcher(),
					Brancher:            pointerToBrancher(),
				},
			},
		},
		Triggers: &pipelinescheduler.ReplaceableSliceOfTriggers{
			Items: []*pipelinescheduler.Trigger{
				&pipelinescheduler.Trigger{
					IgnoreOkToTest: pointerToTrue(),
					JoinOrgURL:     pointerToUUID(),
					OnlyOrgMembers: pointerToTrue(),
					TrustedOrg:     pointerToUUID(),
				},
			},
		},
		ScehdulerAgent: &pipelinescheduler.SchedulerAgent{
			Agent: pointerToUUID(),
		},
		Approve: &pipelinescheduler.Approve{
			RequireSelfApproval: pointerToTrue(),
			LgtmActsAsApprove:   pointerToTrue(),
			IssueRequired:       pointerToTrue(),
			IgnoreReviewState:   pointerToTrue(),
		},
		ExternalPlugins: &pipelinescheduler.ReplaceableSliceOfExternalPlugins{
			Items: []*pipelinescheduler.ExternalPlugin{
				&pipelinescheduler.ExternalPlugin{
					Name:     pointerToUUID(),
					Events:   pointerToReplaceableSliceOfStrings(),
					Endpoint: pointerToUUID(),
				},
			},
		},
		LGTM: &pipelinescheduler.Lgtm{
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

func pointerToReplaceableSliceOfStrings() *pipelinescheduler.ReplaceableSliceOfStrings {
	return &pipelinescheduler.ReplaceableSliceOfStrings{
		Items: []string{
			uuid.New(),
		},
	}
}

func pointerToReplaceableMapOfStringString() *pipelinescheduler.ReplaceableMapOfStringString {
	return &pipelinescheduler.ReplaceableMapOfStringString{
		Items: map[string]string{
			uuid.New(): uuid.New(),
		},
	}
}

func pointerToContextPolicy() *pipelinescheduler.ContextPolicy {
	return &pipelinescheduler.ContextPolicy{
		SkipUnknownContexts:       pointerToTrue(),
		FromBranchProtection:      pointerToTrue(),
		RequiredIfPresentContexts: pointerToReplaceableSliceOfStrings(),
		RequiredContexts:          pointerToReplaceableSliceOfStrings(),
		OptionalContexts:          pointerToReplaceableSliceOfStrings(),
	}
}

func pointerToGlobalProtectionPolicy() *pipelinescheduler.GlobalProtectionPolicy {
	return &pipelinescheduler.GlobalProtectionPolicy{
		ProtectTested:    pointerToTrue(),
		ProtectionPolicy: pointerToProtectionPolicy(),
	}
}

func pointerToProtectionPolicy() *pipelinescheduler.ProtectionPolicy {
	return &pipelinescheduler.ProtectionPolicy{
		Restrictions: &pipelinescheduler.Restrictions{
			Users: pointerToReplaceableSliceOfStrings(),
			Teams: pointerToReplaceableSliceOfStrings(),
		},
		Admins: pointerToTrue(),
		RequiredPullRequestReviews: &pipelinescheduler.ReviewPolicy{
			DismissalRestrictions: &pipelinescheduler.Restrictions{
				Users: pointerToReplaceableSliceOfStrings(),
				Teams: pointerToReplaceableSliceOfStrings(),
			},
		},
		RequiredStatusChecks: &pipelinescheduler.BranchProtectionContextPolicy{
			Strict:   pointerToTrue(),
			Contexts: pointerToReplaceableSliceOfStrings(),
		},
		Protect: pointerToTrue(),
	}
}

func pointerToJobBase() *pipelinescheduler.JobBase {
	return &pipelinescheduler.JobBase{
		Labels:         pointerToReplaceableMapOfStringString(),
		Namespace:      pointerToUUID(),
		Cluster:        pointerToUUID(),
		MaxConcurrency: pointerToRandomNumber(),
		Agent:          pointerToUUID(),
		Name:           pointerToUUID(),
	}
}

func pointerToRegexpChangeMatcher() *pipelinescheduler.RegexpChangeMatcher {
	return &pipelinescheduler.RegexpChangeMatcher{
		RunIfChanged: pointerToUUID(),
	}
}

func pointerToBrancher() *pipelinescheduler.Brancher {
	return &pipelinescheduler.Brancher{
		Branches:     pointerToReplaceableSliceOfStrings(),
		SkipBranches: pointerToReplaceableSliceOfStrings(),
	}
}

// SchedulerFile contains a list of leaf files to build the scheduler from
type SchedulerFile struct {
	// Filenames is the hierachy with the leaf at the right
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
		schedulers := make([]*pipelinescheduler.Scheduler, 0)
		for _, f := range sfs.Filenames {
			bytes, err := ioutil.ReadFile(filepath.Join(baseDir, f))
			assert.NoError(t, err)
			s := pipelinescheduler.Scheduler{}
			err = yaml.Unmarshal(bytes, &s)
			assert.NoError(t, err)
			schedulers = append(schedulers, &s)
		}
		s, err := pipelinescheduler.Build(schedulers)
		assert.NoError(t, err)
		schedulerLeaves = append(schedulerLeaves, &pipelinescheduler.SchedulerLeaf{
			Repo:      sfs.Repo,
			Org:       sfs.Org,
			Scheduler: s,
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
