package prow

import (
	"fmt"

	"k8s.io/test-infra/prow/plugins"

	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/pkg/errors"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
)

// Build takes a list of schedulers and creates a Prow Config from it
func Build(schedulers []*pipelinescheduler.SchedulerLeaf) (*config.Config, *plugins.Configuration,
	error) {
	configResult := config.Config{
		JobConfig:  config.JobConfig{},
		ProwConfig: config.ProwConfig{},
	}
	pluginsResult := plugins.Configuration{}
	for _, scheduler := range schedulers {
		err := buildJobConfig(&configResult.JobConfig, &configResult.ProwConfig, scheduler.Scheduler, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building JobConfig for %v", scheduler)
		}
		err = buildProwConfig(&configResult.ProwConfig, scheduler.Scheduler)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building ProwConfig for %v", scheduler)
		}
		err = buildPlugins(&pluginsResult, scheduler.Scheduler, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building Plugins for %v", scheduler)
		}
	}
	return &configResult, &pluginsResult, nil
}

func buildPlugins(answer *plugins.Configuration, scheduler *pipelinescheduler.Scheduler, orgName string,
	repoName string) error {
	if scheduler.Plugins != nil {
		if answer.Plugins == nil {
			answer.Plugins = make(map[string][]string)
		}
		answer.Plugins[orgSlashRepo(orgName, repoName)] = scheduler.Plugins.Items
	}
	if scheduler.ExternalPlugins != nil {
		if answer.ExternalPlugins == nil {
			answer.ExternalPlugins = make(map[string][]plugins.ExternalPlugin)
		}
		res := make([]plugins.ExternalPlugin, 0)
		for _, src := range scheduler.ExternalPlugins.Items {
			externalPlugin := plugins.ExternalPlugin{}
			err := buildExternalPlugin(&externalPlugin, src)
			if err != nil {
				return errors.Wrapf(err, "building ExternalPlugin for %v", src)
			}
			res = append(res, externalPlugin)
		}
		answer.ExternalPlugins[orgSlashRepo(orgName, repoName)] = res
	}
	if scheduler.Approve != nil {
		if answer.Approve == nil {
			answer.Approve = make([]plugins.Approve, 0)
		}
		approve := plugins.Approve{}
		err := buildApprove(&approve, scheduler.Approve, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building Approve for %v", scheduler.Approve)
		}
		answer.Approve = append(answer.Approve, approve)
	}
	if scheduler.LGTM != nil {
		if answer.Lgtm == nil {
			answer.Lgtm = make([]plugins.Lgtm, 0)
		}
		lgtm := plugins.Lgtm{}
		err := buildLgtm(&lgtm, scheduler.LGTM, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building LGTM for %v", scheduler.LGTM)
		}
		answer.Lgtm = append(answer.Lgtm, lgtm)
	}
	if scheduler.Trigger != nil {
		if answer.Triggers == nil {
			answer.Triggers = make([]plugins.Trigger, 0)
		}
		trigger := plugins.Trigger{}
		err := buildTrigger(&trigger, scheduler.Trigger, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building Triggers for %v", scheduler.Trigger)
		}
		answer.Triggers = append(answer.Triggers, trigger)
	}
	return nil
}

func buildTrigger(answer *plugins.Trigger, trigger *pipelinescheduler.Trigger, orgName string, repoName string) error {
	if trigger.TrustedOrg != nil {
		answer.TrustedOrg = *trigger.TrustedOrg
	} else {
		answer.TrustedOrg = orgName
	}
	if trigger.OnlyOrgMembers != nil {
		answer.OnlyOrgMembers = *trigger.OnlyOrgMembers
	}
	if trigger.JoinOrgURL != nil {
		answer.JoinOrgURL = *trigger.JoinOrgURL
	}
	if trigger.IgnoreOkToTest != nil {
		answer.IgnoreOkToTest = *trigger.IgnoreOkToTest
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
	return nil
}

func buildLgtm(answer *plugins.Lgtm, lgtm *pipelinescheduler.Lgtm, orgName string, repoName string) error {
	if lgtm.StickyLgtmTeam != nil {
		answer.StickyLgtmTeam = *lgtm.StickyLgtmTeam
	}
	if lgtm.ReviewActsAsLgtm != nil {
		answer.ReviewActsAsLgtm = *lgtm.ReviewActsAsLgtm
	}
	if lgtm.StoreTreeHash != nil {
		answer.StoreTreeHash = *lgtm.StoreTreeHash
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
	return nil
}

func buildApprove(answer *plugins.Approve, approve *pipelinescheduler.Approve, orgName string, repoName string) error {
	answer.IgnoreReviewState = approve.IgnoreReviewState
	answer.RequireSelfApproval = approve.RequireSelfApproval
	if approve.IssueRequired != nil {
		answer.IssueRequired = *approve.IssueRequired
	}
	if approve.LgtmActsAsApprove != nil {
		answer.LgtmActsAsApprove = *approve.LgtmActsAsApprove
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
	return nil
}

func buildExternalPlugin(answer *plugins.ExternalPlugin, plugin *pipelinescheduler.ExternalPlugin) error {
	if plugin.Name != nil {
		answer.Name = *plugin.Name
	}
	if plugin.Endpoint != nil {
		answer.Endpoint = *plugin.Endpoint
	}
	if plugin.Events != nil {
		answer.Events = plugin.Events.Items
	}
	return nil
}

func buildProwConfig(prowConfig *config.ProwConfig, scheduler *pipelinescheduler.Scheduler) error {
	if scheduler.Policy != nil {
		err := buildGlobalBranchProtection(&prowConfig.BranchProtection, scheduler.Policy)
		if err != nil {
			return errors.Wrapf(err, "building BranchProtection for %v", scheduler)
		}
	}
	if scheduler.Merger != nil {
		err := buildMerger(&prowConfig.Tide, scheduler.Merger)
		if err != nil {
			return errors.Wrapf(err, "building Merger for %v", scheduler)
		}
	}
	return nil
}

func buildPolicy(answer *config.Policy, policy *pipelinescheduler.ProtectionPolicy) error {
	if policy.Protect != nil {
		answer.Protect = policy.Protect
	}
	if policy.Admins != nil {
		answer.Admins = answer.Admins
	}
	if policy.RequiredStatusChecks != nil {
		if answer.RequiredStatusChecks == nil {
			answer.RequiredStatusChecks = &config.ContextPolicy{}
		}
		err := buildBranchProtectionContextPolicy(answer.RequiredStatusChecks, policy.RequiredStatusChecks)
		if err != nil {
			return errors.Wrapf(err, "building ContextPolicy from %v", policy)
		}
	}
	if policy.RequiredPullRequestReviews != nil {
		if answer.RequiredPullRequestReviews == nil {
			answer.RequiredPullRequestReviews = &config.ReviewPolicy{}
		}
		err := buildRequiredPullRequestReviews(answer.RequiredPullRequestReviews, policy.RequiredPullRequestReviews)
		if err != nil {
			return errors.Wrapf(err, "building RequiredPullRequestReviews from %v", policy)
		}
	}
	if policy.Restrictions != nil {
		if answer.Restrictions == nil {
			answer.Restrictions = &config.Restrictions{}
		}
		err := buildRestrictions(answer.Restrictions, policy.Restrictions)
		if err != nil {
			return errors.Wrapf(err, "building Restrictions from %v", policy)
		}
	}
	return nil
}

func buildBranchProtectionContextPolicy(answer *config.ContextPolicy,
	policy *pipelinescheduler.BranchProtectionContextPolicy) error {
	if policy.Contexts != nil {
		answer.Contexts = policy.Contexts.Items
	}
	if policy.Strict != nil {
		answer.Strict = policy.Strict
	}
	return nil
}

func buildRequiredPullRequestReviews(answer *config.ReviewPolicy, policy *pipelinescheduler.ReviewPolicy) error {
	if policy.Approvals != nil {
		answer.Approvals = policy.Approvals
	}
	if policy.DismissStale != nil {
		answer.DismissStale = policy.DismissStale
	}
	if policy.RequireOwners != nil {
		answer.RequireOwners = policy.RequireOwners
	}
	if policy.DismissalRestrictions != nil {
		if answer.DismissalRestrictions == nil {
			answer.DismissalRestrictions = &config.Restrictions{}
		}
		err := buildRestrictions(answer.DismissalRestrictions, policy.DismissalRestrictions)
		if err != nil {
			return errors.Wrapf(err, "building DismissalRestrictions from %v", policy)
		}
	}
	return nil
}

func buildRestrictions(answer *config.Restrictions, restrictions *pipelinescheduler.Restrictions) error {
	if restrictions.Users != nil {
		answer.Users = restrictions.Users.Items
	}
	if restrictions.Teams != nil {
		answer.Teams = restrictions.Teams.Items
	}
	return nil
}

func buildJobConfig(jobConfig *config.JobConfig, prowConfig *config.ProwConfig,
	scheduler *pipelinescheduler.Scheduler, org string, repo string) error {
	if scheduler.Postsubmits != nil && scheduler.Postsubmits.Items != nil {
		err := buildPostsubmits(jobConfig, scheduler.Postsubmits.Items, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Postsubmits from %v", scheduler)
		}
	}
	if scheduler.Presubmits != nil && scheduler.Presubmits.Items != nil {
		err := buildPresubmits(jobConfig, prowConfig, scheduler.Presubmits.Items, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Presubmits from %v", scheduler)
		}
	}
	return nil
}

func buildPostsubmits(jobConfig *config.JobConfig, items []*pipelinescheduler.Postsubmit, orgName string, repoName string) error {
	if jobConfig.Postsubmits == nil {
		jobConfig.Postsubmits = make(map[string][]config.Postsubmit)
	}
	orgSlashRepo := orgSlashRepo(orgName, repoName)
	for _, postsubmit := range items {
		if _, ok := jobConfig.Postsubmits[orgSlashRepo]; !ok {
			jobConfig.Postsubmits[orgSlashRepo] = make([]config.Postsubmit, 0)
		}
		c := config.Postsubmit{}
		err := buildJobBase(&c.JobBase, postsubmit.JobBase)
		if err != nil {
			return errors.Wrapf(err, "building JobBase from %v", postsubmit.JobBase)
		}
		if postsubmit.Brancher != nil {
			err = buildBrancher(&c.Brancher, postsubmit.Brancher)
			if err != nil {
				return errors.Wrapf(err, "building Brancher from %v", postsubmit.Brancher)
			}
		}
		if postsubmit.RegexpChangeMatcher != nil {
			err = buildRegexChangeMatacher(&c.RegexpChangeMatcher, postsubmit.RegexpChangeMatcher)
			if err != nil {
				return errors.Wrapf(err, "building RegexpChangeMatcher from %v", postsubmit.RegexpChangeMatcher)
			}
		}
		if postsubmit.Report != nil {
			c.Report = !*postsubmit.Report
		}
		if postsubmit.Context != nil {
			c.Context = *postsubmit.Context
		}
		jobConfig.Postsubmits[orgSlashRepo] = append(jobConfig.Postsubmits[orgSlashRepo], c)
	}
	return nil
}

func buildPresubmits(jobConfig *config.JobConfig, prowConfig *config.ProwConfig,
	items []*pipelinescheduler.Presubmit, orgName string, repoName string) error {
	if jobConfig.Presubmits == nil {
		jobConfig.Presubmits = make(map[string][]config.Presubmit)
	}
	orgSlashRepo := orgSlashRepo(orgName, repoName)
	for _, presubmit := range items {
		if _, ok := jobConfig.Presubmits[orgSlashRepo]; !ok {
			jobConfig.Presubmits[orgSlashRepo] = make([]config.Presubmit, 0)
		}
		c := config.Presubmit{}
		err := buildJobBase(&c.JobBase, presubmit.JobBase)
		if err != nil {
			return errors.Wrapf(err, "building JobBase from %v", presubmit.JobBase)
		}
		if presubmit.Brancher != nil {
			err = buildBrancher(&c.Brancher, presubmit.Brancher)
			if err != nil {
				return errors.Wrapf(err, "building Brancher from %v", presubmit.Brancher)
			}
		}
		if presubmit.RegexpChangeMatcher != nil {
			err = buildRegexChangeMatacher(&c.RegexpChangeMatcher, presubmit.RegexpChangeMatcher)
			if err != nil {
				return errors.Wrapf(err, "building RegexpChangeMatcher from %v", presubmit.RegexpChangeMatcher)
			}
		}
		if presubmit.Trigger != nil {
			c.Trigger = *presubmit.Trigger
		}
		if presubmit.RerunCommand != nil {
			c.RerunCommand = *presubmit.RerunCommand
		}
		if presubmit.Optional != nil {
			c.Optional = *presubmit.Optional
		}
		if presubmit.AlwaysRun != nil {
			c.AlwaysRun = *presubmit.AlwaysRun
		}
		if presubmit.Report != nil {
			c.SkipReport = !*presubmit.Report
		}
		if presubmit.Context != nil {
			c.Context = *presubmit.Context
		}
		jobConfig.Presubmits[orgSlashRepo] = append(jobConfig.Presubmits[orgSlashRepo], c)

		if presubmit.Query != nil {
			err := buildQuery(&prowConfig.Tide, presubmit.Query, orgName, repoName)
			if err != nil {
				return errors.Wrapf(err, "building Query from %v", presubmit.Query)
			}
		}
		if presubmit.MergeType != nil {
			mt := github.PullRequestMergeType(*presubmit.MergeType)
			if prowConfig.Tide.MergeType == nil {
				prowConfig.Tide.MergeType = make(map[string]github.PullRequestMergeType)
			}
			prowConfig.Tide.MergeType[orgSlashRepo] = mt
		}
		if presubmit.Policy != nil {
			if presubmit.Policy.ProtectionPolicy != nil {
				err := buildBranchProtection(&prowConfig.BranchProtection, presubmit.Policy.ProtectionPolicy,
					orgName, repoName, "")
				if err != nil {
					return errors.WithStack(err)
				}
			}
			for k, v := range presubmit.Policy.Items {
				err := buildBranchProtection(&prowConfig.BranchProtection, v, orgName, repoName, k)
				if err != nil {
					return errors.WithStack(err)
				}
			}

		}
		if presubmit.ContextPolicy != nil {
			policy := config.TideRepoContextPolicy{
				Branches: make(map[string]config.TideContextPolicy),
			}
			err := buildRepoContextPolicy(&policy, presubmit.ContextPolicy)
			if err != nil {
				return errors.Wrapf(err, "building RepoContextPolicy from %v", presubmit)
			}
			if prowConfig.Tide.ContextOptions.Orgs == nil {
				prowConfig.Tide.ContextOptions.Orgs = make(map[string]config.TideOrgContextPolicy)
			}
			if _, ok := prowConfig.Tide.ContextOptions.Orgs[orgName]; !ok {
				prowConfig.Tide.ContextOptions.Orgs[orgName] = config.TideOrgContextPolicy{
					Repos: make(map[string]config.TideRepoContextPolicy),
				}
			}
			prowConfig.Tide.ContextOptions.Orgs[orgName].Repos[repoName] = policy
		}
		// TODO handle LGTM's here
	}
	return nil
}

func buildGlobalBranchProtection(answer *config.BranchProtection,
	globalProtectionPolicy *pipelinescheduler.GlobalProtectionPolicy) error {
	if globalProtectionPolicy.ProtectTested != nil {
		answer.ProtectTested = *globalProtectionPolicy.ProtectTested
	}
	if globalProtectionPolicy.ProtectionPolicy != nil {
		err := buildBranchProtection(answer, globalProtectionPolicy.ProtectionPolicy, "", "", "")
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func buildBranchProtection(answer *config.BranchProtection,
	protectionPolicy *pipelinescheduler.ProtectionPolicy, orgName string, repoName string, branchName string) error {
	policy := config.Policy{}
	err := buildPolicy(&policy, protectionPolicy)
	if err != nil {
		return errors.Wrapf(err, "building ProtectionPolicy from %v", protectionPolicy)
	}
	if orgName != "" {
		if answer.Orgs == nil {
			answer.Orgs = make(map[string]config.Org)
		}
		if _, ok := answer.Orgs[orgName]; !ok {
			answer.Orgs[orgName] = config.Org{}
		}
		org := answer.Orgs[orgName]
		if repoName != "" {
			if org.Repos == nil {
				org.Repos = make(map[string]config.Repo)
			}
			if _, ok := answer.Orgs[orgName].Repos[repoName]; !ok {
				org.Repos[repoName] = config.Repo{}
			}
			repo := answer.Orgs[orgName].Repos[repoName]
			if branchName != "" {

				if repo.Branches == nil {
					repo.Branches = make(map[string]config.Branch)
				}
				repo.Branches[branchName] = config.Branch{
					Policy: policy,
				}
			} else {
				repo.Policy = policy
			}
			org.Repos[repoName] = repo
		} else {
			org.Policy = policy
		}
		answer.Orgs[orgName] = org
	} else {
		answer.Policy = policy
	}
	return nil
}

func orgSlashRepo(org string, repo string) string {
	if repo == "" {
		return org
	}
	return fmt.Sprintf("%s/%s", org, repo)
}

func buildJobBase(answer *config.JobBase, jobBase *pipelinescheduler.JobBase) error {
	if jobBase.Agent != nil {
		answer.Agent = *jobBase.Agent
	}
	if jobBase.Labels != nil && jobBase.Labels.Items != nil {
		answer.Labels = jobBase.Labels.Items
	}
	if jobBase.MaxConcurrency != nil {
		answer.MaxConcurrency = *jobBase.MaxConcurrency
	}
	if jobBase.Cluster != nil {
		answer.Cluster = *jobBase.Cluster
	}
	if jobBase.Namespace != nil {
		answer.Namespace = jobBase.Namespace
	}
	if jobBase.Name != nil {
		answer.Name = *jobBase.Name
	}
	return nil
}

func buildBrancher(answer *config.Brancher, brancher *pipelinescheduler.Brancher) error {
	if brancher.SkipBranches != nil && brancher.SkipBranches.Items != nil {
		answer.SkipBranches = brancher.SkipBranches.Items
	}
	if brancher.Branches != nil {
		answer.Branches = brancher.Branches.Items
	}
	return nil
}

func buildRegexChangeMatacher(answer *config.RegexpChangeMatcher,
	matcher *pipelinescheduler.RegexpChangeMatcher) error {
	if matcher.RunIfChanged != nil {
		answer.RunIfChanged = *matcher.RunIfChanged
	}
	return nil
}

func buildMerger(answer *config.Tide, merger *pipelinescheduler.Merger) error {
	if merger.SyncPeriod != nil {
		answer.SyncPeriod = *merger.SyncPeriod
	}
	if merger.StatusUpdatePeriod != nil {
		answer.StatusUpdatePeriod = *merger.StatusUpdatePeriod
	}
	if merger.TargetURL != nil {
		answer.TargetURL = *merger.TargetURL
	}
	if merger.PRStatusBaseURL != nil {
		answer.PRStatusBaseURL = *merger.PRStatusBaseURL
	}
	if merger.BlockerLabel != nil {
		answer.BlockerLabel = *merger.BlockerLabel
	}
	if merger.SquashLabel != nil {
		answer.SquashLabel = *merger.SquashLabel
	}
	if merger.MaxGoroutines != nil {
		answer.MaxGoroutines = *merger.MaxGoroutines
	}
	if merger.ContextPolicy != nil {
		err := buildContextPolicy(&answer.ContextOptions.TideContextPolicy, merger.ContextPolicy)
		if err != nil {
			return errors.Wrapf(err, "building ContextPolicy for %v", merger.ContextPolicy)
		}
	}
	return nil
}

func buildRepoContextPolicy(answer *config.TideRepoContextPolicy,
	repoContextPolicy *pipelinescheduler.RepoContextPolicy) error {
	err := buildContextPolicy(&answer.TideContextPolicy, repoContextPolicy.ContextPolicy)
	if err != nil {
		return errors.Wrapf(err, "building ContextPolicy for %v", repoContextPolicy)
	}
	if repoContextPolicy.Branches != nil {
		for branch, policy := range repoContextPolicy.Branches.Items {
			tidePolicy := config.TideContextPolicy{}
			err := buildContextPolicy(&tidePolicy, policy)
			if err != nil {
				return errors.Wrapf(err, "building ContextPolicy for %v", policy)
			}
			answer.Branches[branch] = tidePolicy
		}
	}
	return nil
}

func buildContextPolicy(answer *config.TideContextPolicy,
	contextOptions *pipelinescheduler.ContextPolicy) error {
	if contextOptions.SkipUnknownContexts != nil {
		answer.SkipUnknownContexts = contextOptions.SkipUnknownContexts
	}
	if contextOptions.FromBranchProtection != nil {
		answer.FromBranchProtection = contextOptions.FromBranchProtection
	}
	if contextOptions.RequiredIfPresentContexts != nil {
		answer.RequiredIfPresentContexts = contextOptions.RequiredIfPresentContexts.Items
	}
	if contextOptions.RequiredContexts != nil {
		answer.RequiredContexts = contextOptions.RequiredContexts.Items
	}
	if contextOptions.OptionalContexts != nil {
		answer.OptionalContexts = contextOptions.OptionalContexts.Items
	}
	return nil
}

func buildQuery(answer *config.Tide, query *pipelinescheduler.Query, org string, repo string) error {
	if answer.Queries == nil {
		answer.Queries = config.TideQueries{}
	}
	tideQuery := config.TideQuery{
		Repos: []string{orgSlashRepo(org, repo)},
	}
	if query.ExcludedBranches != nil {
		tideQuery.ExcludedBranches = query.ExcludedBranches.Items
	}
	if query.IncludedBranches != nil {
		tideQuery.IncludedBranches = query.IncludedBranches.Items
	}
	if query.Labels != nil {
		tideQuery.Labels = query.Labels.Items
	}
	if query.MissingLabels != nil {
		tideQuery.MissingLabels = query.MissingLabels.Items
	}
	if query.Milestone != nil {
		tideQuery.Milestone = *query.Milestone
	}
	if query.ReviewApprovedRequired != nil {
		tideQuery.ReviewApprovedRequired = *query.ReviewApprovedRequired
	}
	answer.Queries = append(answer.Queries, tideQuery)
	return nil
}
