package draftpath

import (
	"path/filepath"
)

// Home describes the location of a CLI configuration.
//
// This helper builds paths relative to a Draft Home directory.
type Home string

// Path returns Home with elements appended.
func (h Home) Path(elem ...string) string {
	p := []string{h.String()}
	p = append(p, elem...)
	return filepath.Join(p...)
}

// Config returns the path to the Draft config file.
func (h Home) Config() string {
	return h.Path("config.toml")
}

// Packs returns the path to the Draft starter packs.
func (h Home) Packs() string {
	return h.Path("packs")
}

// Logs returns the path to the Draft logs.
func (h Home) Logs() string {
	return h.Path("logs")
}

// Plugins returns the path to the Draft plugins.
func (h Home) Plugins() string {
	return h.Path("plugins")
}

// String returns Home as a string.
//
// Implements fmt.Stringer.
func (h Home) String() string {
	return string(h)
}
