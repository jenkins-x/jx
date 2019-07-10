package opts

import (
	"github.com/jenkins-x/jx/pkg/version"
)

// VersionResolver resolves versions of charts, packages or docker images
type VersionResolver struct {
	VersionsDir string
}

// CreateVersionResolver creates a new VersionResolver service
func (o *CommonOptions) CreateVersionResolver(repo string, gitRef string) (*VersionResolver, error) {
	versionsDir, err := o.CloneJXVersionsRepo(repo, gitRef)
	if err != nil {
		return nil, err
	}
	return &VersionResolver{
		VersionsDir: versionsDir,
	}, nil
}

// ResolveDockerImage ensures the given docker image has a valid version if there is one in the version stream
func (v *VersionResolver) ResolveDockerImage(image string) (string, error) {
	return version.ResolveDockerImage(v.VersionsDir, image)
}

// StableVersion returns the stable version of the given kind name
func (v *VersionResolver) StableVersion(kind version.VersionKind, name string) (*version.StableVersion, error) {
	return version.LoadStableVersion(v.VersionsDir, kind, name)
}

// StableVersionNumber returns the stable version number of the given kind name
func (v *VersionResolver) StableVersionNumber(kind version.VersionKind, name string) (string, error) {
	return version.LoadStableVersionNumber(v.VersionsDir, kind, name)
}

// GetVersionNumber returns the version number for the given kind and name or blank string if there is no locked version
func (o *CommonOptions) GetVersionNumber(kind version.VersionKind, name, repo string, gitRef string) (string, error) {
	versioner, err := o.CreateVersionResolver(repo, gitRef)
	if err != nil {
		return "", err
	}
	return versioner.StableVersionNumber(kind, name)
}
