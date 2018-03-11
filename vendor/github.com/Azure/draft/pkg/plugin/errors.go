package plugin

import "errors"

var (
	// ErrMissingMetadata indicates that plugin.yaml is missing.
	ErrMissingMetadata = errors.New("plugin metadata (plugin.yaml) missing")
	// ErrExists indicates that a plugin already exists
	ErrExists = errors.New("plugin already exists")
	// ErrDoesNotExist indicates that a plugin does not exist
	ErrDoesNotExist = errors.New("plugin does not exist")
	// ErrHomeMissing indicates that the directory expected to contain plugins does not exist
	ErrHomeMissing = errors.New(`plugin home "$(draft home)/plugins" does not exist`)
	// ErrMissingSource indicates that information about the source of the plugin was not found
	ErrMissingSource = errors.New("cannot get information about plugin source")
	// ErrRepoDirty indicates that the plugin repo was modified
	ErrRepoDirty = errors.New("plugin repo was modified")
	// ErrVersionDoesNotExist indicates that the request version does not exist
	ErrVersionDoesNotExist = errors.New("requested version does not exist")
)
