package repo

import "errors"

var (
	ErrExists       = errors.New("pack repo already exists")
	ErrDoesNotExist = errors.New("pack repo does not exist")
	// ErrHomeMissing indicates that the packs dir is missing.
	ErrHomeMissing         = errors.New(`pack repo home "$(draft home)/packs" does not exist`)
	ErrMissingSource       = errors.New("cannot get information about pack repo source")
	ErrRepoDirty           = errors.New("pack repo was modified")
	ErrVersionDoesNotExist = errors.New("requested version does not exist")
)
