package cmd

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
)

func (o *CommonOptions) CreateIssueTrackerAuthConfigService(dir string) (auth.AuthConfigService, error) {
	var err error
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			o.warnf("Could not find the current directory %s\n", err)
			return o.Factory.CreateIssueTrackerAuthConfigService("")
		}
	}
	pc, _, err := config.LoadProjectConfig(dir)
	if err != nil {
		o.warnf("Could not load the Project's Jenkins X configuration from dir %s due to %s\n", dir, err)
		return o.Factory.CreateIssueTrackerAuthConfigService("")
	}
	return o.CreateIssueTrackerAuthConfigServiceFromConfig(pc)
}

func (o *CommonOptions) CreateIssueTrackerAuthConfigServiceFromConfig(pc *config.ProjectConfig) (auth.AuthConfigService, error) {
	issueURL := ""
	if pc != nil {
		it := pc.IssueTracker
		if it != nil {
			issueURL = it.URL
		}
	}
	return o.Factory.CreateIssueTrackerAuthConfigService(issueURL)
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
				authConfigSvc, err := o.Factory.CreateIssueTrackerAuthConfigService(it.URL)
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
