package config

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/test-infra/prow/config"
	"strings"
)

// AddRepoToBranchProtection adds a repository to the Branch Protection section of a prow config
func AddRepoToBranchProtection(bp *config.BranchProtection, repoSpec string, context string, kind Kind) error {
	bp.ProtectTested = true
	if bp.Orgs == nil {
		bp.Orgs = make(map[string]config.Org, 0)
	}
	s := strings.Split(repoSpec, "/")
	if len(s) != 2 {
		return fmt.Errorf("%s is not of the format org/repo", repoSpec)
	}
	requiredOrg := s[0]
	requiredRepo := s[1]
	if _, ok := bp.Orgs[requiredOrg]; !ok {
		bp.Orgs[requiredOrg] = config.Org{
			Repos: make(map[string]config.Repo, 0),
		}
	}
	if _, ok := bp.Orgs[requiredOrg].Repos[requiredRepo]; !ok {
		bp.Orgs[requiredOrg].Repos[requiredRepo] = config.Repo{
			Policy: config.Policy{
				RequiredStatusChecks: &config.ContextPolicy{},
			},
		}

	}
	if bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts == nil {
		bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts = make([]string, 0)
	}
	contexts := bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts
	switch kind {
	case Application:
		if !util.Contains(contexts, ServerlessJenkins) {
			contexts = append(contexts, ServerlessJenkins)
		}
	case Environment:
		if !util.Contains(contexts, PromotionBuild) {
			contexts = append(contexts, PromotionBuild)
		}
	case Protection:
		if !util.Contains(contexts, ComplianceCheck) {
			contexts = append(contexts, context)
		}
	default:
		return fmt.Errorf("unknown Prow config kind %s", kind)
	}
	bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts = contexts
	return nil
}

// GetAllBranchProtectionContexts gets all the contexts that have branch protection for a repo
func GetAllBranchProtectionContexts(org string, repo string, prowConfig *config.Config) ([]string, error) {
	prowOrg, ok := prowConfig.BranchProtection.Orgs[org]
	if !ok {
		prowOrg = config.Org{}
	}
	if prowOrg.Repos == nil {
		prowOrg.Repos = make(map[string]config.Repo, 0)
	}
	prowRepo, ok := prowOrg.Repos[repo]
	if !ok {
		prowRepo = config.Repo{}
	}
	if prowRepo.RequiredStatusChecks == nil {
		prowRepo.RequiredStatusChecks = &config.ContextPolicy{}
	}
	return prowRepo.RequiredStatusChecks.Contexts, nil
}

func GetBranchProtectionContexts(org string, repo string, prowConfig *config.Config) ([]string, error) {
	result := make([]string, 0)
	contexts, err := GetAllBranchProtectionContexts(org, repo, prowConfig)
	if err != nil {
		return result, err
	}
	for _, c := range contexts {
		if c != ServerlessJenkins && c != PromotionBuild {
			result = append(result, c)
		}
	}
	return result, nil
}
