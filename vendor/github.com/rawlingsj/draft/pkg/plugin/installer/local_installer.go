package installer

import (
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
)

// LocalInstaller installs plugins from the filesystem
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

// Install creates a symlink to the plugin directory in $DRAFT_HOME
func (i *LocalInstaller) Install() error {
	if !isPlugin(i.Source) {
		return plugin.ErrMissingMetadata
	}

	src, err := filepath.Abs(i.Source)
	if err != nil {
		return err
	}

	return i.link(src)
}

// Update updates a local repository
func (i *LocalInstaller) Update() error {
	debug("local repository is auto-updated")
	return nil
}
