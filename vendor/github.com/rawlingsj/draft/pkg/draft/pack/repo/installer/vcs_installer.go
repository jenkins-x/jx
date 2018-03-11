package installer

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver"
	"github.com/Masterminds/vcs"
	"k8s.io/helm/pkg/plugin/cache"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/osutil"
	"github.com/Azure/draft/pkg/plugin/installer"
)

//VCSInstaller installs packs from a remote repository
type VCSInstaller struct {
	Repo    vcs.Repo
	Version string
	base
}

// NewVCSInstaller creates a new VCSInstaller.
func NewVCSInstaller(source, version string, home draftpath.Home) (*VCSInstaller, error) {
	// create a system safe cache key
	key, err := cache.Key(source)
	if err != nil {
		return nil, err
	}
	cachedpath := home.Path("cache", "packs", key)
	repo, err := vcs.NewRepo(source, cachedpath)
	if err != nil {
		return nil, err
	}

	i := &VCSInstaller{
		Repo:    repo,
		Version: version,
		base:    newBase(source, home),
	}
	return i, err
}

// Path is where the pack repo will be symlinked to.
func (i *VCSInstaller) Path() string {
	u, err := url.Parse(i.Source)
	if err == nil {
		return filepath.Join(i.DraftHome.Packs(), u.Host, u.Path)
	}
	return filepath.Join(i.DraftHome.Packs(), filepath.Base(i.Source))
}

// Install clones a remote repository and creates a symlink to the pack repo directory in DRAFT_HOME
//
// Implements Installer
func (i *VCSInstaller) Install() error {
	if err := i.sync(i.Repo); err != nil {
		return err
	}

	ref, err := i.solveVersion(i.Repo)
	if err != nil {
		return err
	}

	if err := i.setVersion(i.Repo, ref); err != nil {
		return err
	}

	if !isPackRepo(i.Repo.LocalPath()) {
		return repo.ErrHomeMissing
	}

	if err := os.MkdirAll(path.Dir(i.Path()), 0755); err != nil {
		return err
	}

	return osutil.SymlinkWithFallback(i.Repo.LocalPath(), i.Path())
}

// Update updates a remote repository
func (i *VCSInstaller) Update() error {
	if i.Repo.IsDirty() {
		return repo.ErrRepoDirty
	}
	if err := i.Repo.Update(); err != nil {
		return err
	}
	if !isPackRepo(i.Repo.LocalPath()) {
		return repo.ErrHomeMissing
	}
	return nil
}

func existingVCSRepo(location string, home draftpath.Home) (installer.Installer, error) {
	repo, err := vcs.NewRepo("", location)
	if err != nil {
		return nil, err
	}
	i := &VCSInstaller{
		Repo: repo,
		base: newBase(repo.Remote(), home),
	}

	return i, err
}

// Filter a list of versions to only included semantic versions. The response
// is a mapping of the original version to the semantic version.
func getSemVers(refs []string) []*semver.Version {
	var sv []*semver.Version
	for _, r := range refs {
		if v, err := semver.NewVersion(r); err == nil {
			sv = append(sv, v)
		}
	}
	return sv
}

// setVersion attempts to checkout the version
func (i *VCSInstaller) setVersion(repo vcs.Repo, ref string) error {
	return repo.UpdateVersion(ref)
}

func (i *VCSInstaller) solveVersion(rp vcs.Repo) (string, error) {
	if i.Version == "" {
		return "", nil
	}

	if rp.IsReference(i.Version) {
		return i.Version, nil
	}

	// Create the constraint first to make sure it's valid before
	// working on the repo.
	constraint, err := semver.NewConstraint(i.Version)
	if err != nil {
		return "", err
	}

	// Get the tags
	refs, err := rp.Tags()
	if err != nil {
		return "", err
	}

	// Convert and filter the list to semver.Version instances
	semvers := getSemVers(refs)

	// Sort semver list
	sort.Sort(sort.Reverse(semver.Collection(semvers)))
	for _, v := range semvers {
		if constraint.Check(v) {
			// If the constraint passes get the original reference
			ver := v.Original()
			return ver, nil
		}
	}

	return "", repo.ErrVersionDoesNotExist
}

// sync will clone or update a remote repo.
func (i *VCSInstaller) sync(repo vcs.Repo) error {

	if _, err := os.Stat(repo.LocalPath()); os.IsNotExist(err) {
		return repo.Get()
	}
	return repo.Update()
}
