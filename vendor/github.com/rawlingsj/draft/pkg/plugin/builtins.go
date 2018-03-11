package plugin

import "github.com/Azure/draft/pkg/version"

// Builtin contains metadata to the built-in plugins. Used to install/uninstall a plugin.
type Builtin struct {
	Name    string
	URL     string
	Version string
}

// Builtins fetches all built-in plugins.
func Builtins() []*Builtin {
	var packRepoVersion string
	// canary draft releases should always test the latest version of the plugin.
	if version.Release != "canary" {
		packRepoVersion = "v0.3.1"
	}
	return []*Builtin{
		{
			Name:    "pack-repo",
			URL:     "https://github.com/Azure/draft-pack-repo",
			Version: packRepoVersion,
		},
	}
}
