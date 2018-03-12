package plugin

import "errors"

var (
	// ErrMissingMetadata indicates that plugin.yaml is missing.
	ErrMissingMetadata     = errors.New("plugin metadata (plugin.yaml) missing")
	ErrExists              = errors.New("plugin already exists")
	ErrDoesNotExist        = errors.New("plugin does not exist")
	ErrHomeMissing         = errors.New(`plugin home "$(draft home)/plugins" does not exist`)
	ErrMissingSource       = errors.New("cannot get information about plugin source")
	ErrRepoDirty           = errors.New("plugin repo was modified")
	ErrVersionDoesNotExist = errors.New("requested version does not exist")
)
