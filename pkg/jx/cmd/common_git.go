package cmd

import (
	"github.com/jenkins-x/jx/pkg/gits"
)

// createGitProvider creates a git from the given directory
func (o *CommonOptions) createGitProvider(dir string) (*gits.GitRepositoryInfo, gits.GitProvider, error) {
	gitDir, gitConfDir, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return nil, nil, err
	}
	if gitDir == "" || gitConfDir == "" {
		o.warnf("No git directory could be found from dir %s\n", dir)
		return nil, nil, nil
	}

	gitUrl, err := gits.DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, nil, err
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		return nil, nil, err
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return nil, nil, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	gitProvider, err := gitInfo.CreateProvider(authConfigSvc, gitKind)
	if err != nil {
		return nil, nil, err
	}
	return gitInfo, gitProvider, nil
}
