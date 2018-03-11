package installer // import "github.com/Azure/draft/pkg/plugin/installer"

import (
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/osutil"
)

type base struct {
	// Source is the reference to a plugin
	Source string

	// DraftHome is the $DRAFT_HOME directory
	DraftHome draftpath.Home
}

func newBase(source string, home draftpath.Home) base {
	return base{source, home}
}

// link creates a symlink from the plugin source to $DRAFT_HOME
func (b *base) link(from string) error {
	origin, err := filepath.Abs(from)
	if err != nil {
		return err
	}
	dest, err := filepath.Abs(b.Path())
	if err != nil {
		return err
	}
	debug("symlinking %s to %s", origin, dest)
	return osutil.SymlinkWithFallback(origin, dest)
}

// Path is where the plugin will be symlinked to.
func (b *base) Path() string {
	if b.Source == "" {
		return ""
	}
	return filepath.Join(b.DraftHome.Plugins(), filepath.Base(b.Source))
}
