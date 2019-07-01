package opts

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
)

// CreateIssueTrackerConfigService creates auth config service for issue tracker
func (o *CommonOptions) CreateIssueTrackerConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateIssueTrackerConfigService(auth.AutoConfigKind)
}

// CreateIssueProvider creates a issues provider from a give directory which contains a git repository
func (o *CommonOptions) CreateIssueProviderFromDir(dir string) (issues.IssueProvider, error) {
	gitDir, gitConfDir, err := o.Git().FindGitConfigDir(dir)
	if err != nil {
		return nil, fmt.Errorf("no issue tracker configured for this project and cannot find the .git directory: %s", err)
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
				cs, err := o.CreateIssueTrackerConfigService()
				if err != nil {
					return nil, err
				}
				cfg, err := cs.Config()
				if err != nil {
					return nil, err
				}
				server, err := cfg.GetServer(it.URL)
				if err != nil {
					return nil, err
				}
				return issues.CreateIssueProvider(it.Kind, server, it.Project, o.BatchMode, o.Git())
			}
		}
	}
	if gitConfDir == "" {
		return nil, fmt.Errorf("no issue tracker configured and no git directory could be found from dir %q\n", dir)
	}
	gitURL, err := o.Git().DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, fmt.Errorf("no issue tracker configured and could not find the upstream git URL for dir %s, due to: %s\n", dir, err)
	}
	gitProvider, err := o.factory.CreateGitProvider(gitURL, auth.AutoConfigKind, o.Git())
	if err != nil {
		return nil, err
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	return issues.CreateGitIssueProvider(gitProvider, gitInfo.Organisation, gitInfo.Name)
}
