package testhelpers

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
				OptionalContexts:          PointerToReplaceableSliceOfStrings(),
				RequiredContexts:          PointerToReplaceableSliceOfStrings(),
				RequiredIfPresentContexts: PointerToReplaceableSliceOfStrings(),
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
				{
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
						Labels:                 PointerToReplaceableSliceOfStrings(),
						ExcludedBranches:       PointerToReplaceableSliceOfStrings(),
						IncludedBranches:       PointerToReplaceableSliceOfStrings(),
						MissingLabels:          PointerToReplaceableSliceOfStrings(),
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
				{
					Report:              pointerToTrue(),
					Context:             pointerToUUID(),
					JobBase:             pointerToJobBase(),
					RegexpChangeMatcher: pointerToRegexpChangeMatcher(),
					Brancher:            pointerToBrancher(),
				},
			},
		},
		Trigger: &pipelinescheduler.Trigger{
			IgnoreOkToTest: pointerToTrue(),
			JoinOrgURL:     pointerToUUID(),
			OnlyOrgMembers: pointerToTrue(),
			TrustedOrg:     pointerToUUID(),
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
				{
					Name:     pointerToUUID(),
					Events:   PointerToReplaceableSliceOfStrings(),
					Endpoint: pointerToUUID(),
				},
			},
		},
		LGTM: &pipelinescheduler.Lgtm{
			StoreTreeHash:    pointerToTrue(),
			ReviewActsAsLgtm: pointerToTrue(),
			StickyLgtmTeam:   pointerToUUID(),
		},
		Plugins: PointerToReplaceableSliceOfStrings(),
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

// PointerToReplaceableSliceOfStrings creaters a ReplaceableSliceOfStrings and returns its pointer
func PointerToReplaceableSliceOfStrings() *pipelinescheduler.ReplaceableSliceOfStrings {
	return &pipelinescheduler.ReplaceableSliceOfStrings{
		Items: []string{
			uuid.New(),
		},
	}
}

// PointerToReplaceableMapOfStringString creaters a ReplaceableMapOfStringString and returns its pointer
func PointerToReplaceableMapOfStringString() *pipelinescheduler.ReplaceableMapOfStringString {
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
		RequiredIfPresentContexts: PointerToReplaceableSliceOfStrings(),
		RequiredContexts:          PointerToReplaceableSliceOfStrings(),
		OptionalContexts:          PointerToReplaceableSliceOfStrings(),
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
			Users: PointerToReplaceableSliceOfStrings(),
			Teams: PointerToReplaceableSliceOfStrings(),
		},
		Admins: pointerToTrue(),
		RequiredPullRequestReviews: &pipelinescheduler.ReviewPolicy{
			DismissalRestrictions: &pipelinescheduler.Restrictions{
				Users: PointerToReplaceableSliceOfStrings(),
				Teams: PointerToReplaceableSliceOfStrings(),
			},
		},
		RequiredStatusChecks: &pipelinescheduler.BranchProtectionContextPolicy{
			Strict:   pointerToTrue(),
			Contexts: PointerToReplaceableSliceOfStrings(),
		},
		Protect: pointerToTrue(),
	}
}

func pointerToJobBase() *pipelinescheduler.JobBase {
	return &pipelinescheduler.JobBase{
		Labels:         PointerToReplaceableMapOfStringString(),
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
		Branches:     PointerToReplaceableSliceOfStrings(),
		SkipBranches: PointerToReplaceableSliceOfStrings(),
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
