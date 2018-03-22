package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
)

func (o *CommonOptions) createIssueProvider(dir string) (issues.IssueProvider, error) {
	pc, _, err := config.LoadProjectConfig(dir)
	if err != nil {
		return nil, err
	}
	if pc != nil {
		it := pc.IssueTracker
		if it != nil {
			if it.URL != "" && it.Kind != "" {
				authConfigSvc, err := o.Factory.CreateIssueTrackerAuthConfigService()
				if err != nil {
					return nil, err
				}
				config := authConfigSvc.Config()
				server := config.GetOrCreateServer(it.URL)
				return issues.CreateIssueProvider(it.Kind, server)
			}
		}
	}

	_, gitConfDir, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return nil, fmt.Errorf("No issue tracker configured for this project and cannot find the .git directory: %s", err)
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
