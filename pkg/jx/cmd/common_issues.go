package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
	"github.com/jenkins-x/jx/pkg/kube"
)

func (o *CommonOptions) CreateIssueTrackerAuthConfigService() (auth.AuthConfigService, error) {
	secrets, err := o.Factory.LoadPipelineSecrets(kube.ValueKindIssue, "")
	if err != nil {
		o.warnf("The current user cannot query pipeline issue tracker secrets: %s", err)
	}
	return o.Factory.CreateIssueTrackerAuthConfigService(secrets)
}

func (o *CommonOptions) errorCreateIssueTrackerAuthConfigService(parentError error) (auth.AuthConfigService, error) {
	answer, err := o.Factory.CreateIssueTrackerAuthConfigService(nil)
	if err == nil {
		return answer, parentError
	}
	return answer, err
}

func (o *CommonOptions) createIssueProvider(dir string) (issues.IssueProvider, error) {
	gitDir, gitConfDir, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return nil, fmt.Errorf("No issue tracker configured for this project and cannot find the .git directory: %s", err)
	}
	pc, _, err := config.LoadProjectConfig(dir)
	if err != nil {
		return nil, err
	}
	if pc != nil && pc.IssueTracker == nil {
		pc, _, err = config.LoadProjectConfig(gitDir)
		if err != nil {
			return nil, err
		}
	}
	if pc != nil {
		it := pc.IssueTracker
		if it != nil {
			if it.URL != "" && it.Kind != "" {
				authConfigSvc, err := o.CreateIssueTrackerAuthConfigService()
				if err != nil {
					return nil, err
				}
				config := authConfigSvc.Config()
				server := config.GetOrCreateServer(it.URL)
				userAuth, err := config.PickServerUserAuth(server, "user to access the issue tracker", o.BatchMode)
				if err != nil {
					return nil, err
				}
				return issues.CreateIssueProvider(it.Kind, server, userAuth, it.Project, o.BatchMode)
			}
		}
	}

	if gitConfDir == "" {
		return nil, fmt.Errorf("No issue tracker configured and no git directory could be found from dir %s\n", dir)
	}
	gitUrl, err := gits.DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, fmt.Errorf("No issue tracker configured and could not find the upstream git URL for dir %s, due to: %s\n", dir, err)
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		return nil, err
	}
	gitProvider, err := o.gitProviderForURL(gitUrl, "user name to use for authenticating with git issues")
	if err != nil {
		return nil, err
	}
	return issues.CreateGitIssueProvider(gitProvider, gitInfo.Organisation, gitInfo.Name)
}
