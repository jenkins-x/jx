package installer

import (
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/osutil"
)

// LocalInstaller installs pack repos from the filesystem
type LocalInstaller struct {
	base
}

// NewLocalInstaller creates a new LocalInstaller
func NewLocalInstaller(source string, home draftpath.Home) (*LocalInstaller, error) {

	i := &LocalInstaller{
		base: newBase(source, home),
	}

	return i, nil
}

// Path is where the pack repo will be symlinked to.
func (i *LocalInstaller) Path() string {
	if i.Source == "" {
		return ""
	}
	return filepath.Join(i.DraftHome.Packs(), filepath.Base(i.Source))
}

// Install creates a symlink to the pack repo directory in $DRAFT_HOME
func (i *LocalInstaller) Install() error {
	if !isPackRepo(i.Source) {
		return repo.ErrHomeMissing
	}

	src, err := filepath.Abs(i.Source)
	if err != nil {
		return err
	}

	return osutil.SymlinkWithFallback(src, i.Path())
}

// Update updates a local repository
func (i *LocalInstaller) Update() error {
	return nil
}
