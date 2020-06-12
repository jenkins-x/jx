package versionstream

// Streamer contains the functions related to version stream
//go:generate pegomock generate github.com/jenkins-x/jx/v2/pkg/versionstream Streamer -o mocks/streamer.go
type Streamer interface {
	ResolveDockerImage(image string) (string, error)
	StableVersion(kind VersionKind, name string) (*StableVersion, error)
	StableVersionNumber(kind VersionKind, name string) (string, error)
	ResolveGitVersion(gitURL string) (string, error)
	VerifyPackages(packages map[string]string) error
	VerifyPackage(name string, currentVersion string) error
	GetRepositoryPrefixes() (*RepositoryPrefixes, error)
	GetVersionsDir() string
}
