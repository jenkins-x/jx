package plugin

// Builtin contains metadata to the built-in plugins. Used to install/uninstall a plugin.
type Builtin struct {
	Name    string
	URL     string
	Version string
}

// Builtins fetches all built-in plugins.
func Builtins() []*Builtin {
	packRepoVersion := "0.4.2" // Can set this to canary to test latest version of plugin

	return []*Builtin{
		{
			Name:    "pack-repo",
			URL:     "https://github.com/draftcreate/draft-pack-repo",
			Version: packRepoVersion,
		},
	}
}
