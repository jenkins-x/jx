package cmd

import (
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
)

// createGitProvider creates a git from the given directory
func (o *CommonOptions) createGitProvider(dir string) (*gits.GitRepositoryInfo, gits.GitProvider, issues.IssueProvider, error) {
	gitDir, gitConfDir, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	if gitDir == "" || gitConfDir == "" {
		o.warnf("No git directory could be found from dir %s\n", dir)
		return nil, nil, nil, nil
	}

	gitUrl, err := gits.DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, nil, nil, err
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		return nil, nil, nil, err
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return gitInfo, nil, nil, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	gitProvider, err := gitInfo.CreateProvider(authConfigSvc, gitKind)
	if err != nil {
		return gitInfo, gitProvider, nil, err
	}
	tracker, err := o.createIssueProvider(dir)
	if err != nil {
		return gitInfo, gitProvider, tracker, err
	}
	return gitInfo, gitProvider, tracker, nil
}
