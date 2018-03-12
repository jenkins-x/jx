package installer

import (
	"os"
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/plugin/installer"
)

type base struct {
	// Source is the reference to a pack repo
	Source string

	// DraftHome is the $DRAFT_HOME directory
	DraftHome draftpath.Home
}

func newBase(source string, home draftpath.Home) base {
	return base{source, home}
}

// isPackRepo checks if the directory contains a packs directory.
func isPackRepo(dirname string) bool {
	fi, err := os.Stat(filepath.Join(dirname, "packs"))
	return err == nil && fi.IsDir()
}

// isLocalReference checks if the source exists on the filesystem.
func isLocalReference(source string) bool {
	_, err := os.Stat(source)
	return err == nil
}

// Install installs a pack repo to $DRAFT_HOME
func Install(i installer.Installer) error {
	if _, pathErr := os.Stat(i.Path()); !os.IsNotExist(pathErr) {
		return repo.ErrExists
	}

	return i.Install()
}

// Update updates a pack repo in $DRAFT_HOME.
func Update(i installer.Installer) error {
	if _, pathErr := os.Stat(i.Path()); os.IsNotExist(pathErr) {
		return repo.ErrDoesNotExist
	}

	return i.Update()
}

// FindSource determines the correct Installer for the given source.
func FindSource(location string, home draftpath.Home) (installer.Installer, error) {
	installer, err := existingVCSRepo(location, home)
	if err != nil && err.Error() == "Cannot detect VCS" {
		return installer, repo.ErrMissingSource
	}
	return installer, err
}

// New determines and returns the correct Installer for the given source
func New(source, version string, home draftpath.Home) (installer.Installer, error) {
	if isLocalReference(source) {
		return NewLocalInstaller(source, home)
	}

	return NewVCSInstaller(source, version, home)
}
