package version

// Version contains the semver release, git commit, and git tree state.
type Version struct {
	SemVer       string `json:"semver"`
	GitCommit    string `json:"git-commit"`
	GitTreeState string `json:"git-tree-state"`
}

func (v *Version) String() string {
	return v.SemVer
}

var (
	// Release is the current release of Draft.
	// Update this whenever making a new release.
	// The release is of the format Major.Minor.Patch[-Prerelease][+BuildMetadata]
	//
	// If it is a development build, the release name is called "canary".
	//
	// Increment major number for new feature additions and behavioral changes.
	// Increment minor number for bug fixes and performance enhancements.
	// Increment patch number for critical fixes to existing releases.
	Release = "canary"

	// BuildMetadata is extra build time data
	BuildMetadata = ""
	// GitCommit is the git sha1
	GitCommit = ""
	// GitTreeState is the state of the git tree
	GitTreeState = ""
)

// getVersion returns the semver string of the version
func getVersion() string {
	if BuildMetadata == "" {
		return Release
	}
	return Release + "+" + BuildMetadata
}

// New returns the semver interpretation of the version.
func New() *Version {
	return &Version{
		SemVer:       getVersion(),
		GitCommit:    GitCommit,
		GitTreeState: GitTreeState,
	}
}
