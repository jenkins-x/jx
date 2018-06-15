package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
)

type BranchPatterns struct {
	DefaultBranchPattern string
	ForkBranchPattern    string
}

const (
	defaultBuildPackURL = "https://jenkins-x/draft-packs.git"
	defaultBuildPackRef = "master"
	defaultHelmBin      = "helm"
)

// TeamSettings returns the team settings
func (o *CommonOptions) TeamSettings() (*v1.TeamSettings, error) {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return nil, err
	}

	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return nil, err
	}
	if env == nil {
		return nil, fmt.Errorf("No Development environment found for namespace %s", ns)
	}

	teamSettings := &env.Spec.TeamSettings
	if teamSettings.BuildPackURL == "" {
		teamSettings.BuildPackURL = defaultBuildPackURL
	}
	if teamSettings.BuildPackRef == "" {
		teamSettings.BuildPackRef = defaultBuildPackRef
	}
	return teamSettings, nil
}

// TeamBranchPatterns returns the team branch patterns used to enable CI/CD on branches when creating/importing projects
func (o *CommonOptions) TeamBranchPatterns() (*BranchPatterns, error) {
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return nil, err
	}

	branchPatterns := teamSettings.BranchPatterns
	if branchPatterns == "" {
		branchPatterns = defaultBranchPatterns
	}

	forkBranchPatterns := teamSettings.ForkBranchPatterns
	if forkBranchPatterns == "" {
		forkBranchPatterns = defaultForkBranchPatterns
	}

	return &BranchPatterns{
		DefaultBranchPattern: branchPatterns,
		ForkBranchPattern:    forkBranchPatterns,
	}, nil
}

// TeamHelmBin returns the helm binary used for a team
func (o *CommonOptions) TeamHelmBin() (string, error) {
	helmBin := defaultHelmBin
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return helmBin, err
	}

	helmBin = teamSettings.HelmBinary
	if helmBin == "" {
		helmBin = defaultHelmBin
	}
	return helmBin, nil
}

// ModifyDevEnvironment modifies the development environment settings
func (o *CommonOptions) ModifyDevEnvironment(callback func(env *v1.Environment) error) error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return err
	}

	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return err
	}
	if env == nil {
		return fmt.Errorf("No Development environment found for namespace %s", ns)
	}
	return o.modifyDevEnvironment(jxClient, ns, callback)
}
