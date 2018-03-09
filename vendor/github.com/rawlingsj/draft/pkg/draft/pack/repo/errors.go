package repo

import "errors"

var (
	// ErrExists indicates that the pack repo already exists
	ErrExists = errors.New("pack repo already exists")
	// ErrDoesNotExist indicates that the pack repo does not exist
	ErrDoesNotExist = errors.New("pack repo does not exist")
	// ErrHomeMissing indicates that the packs dir is missing.
	ErrHomeMissing = errors.New(`pack repo home "$(draft home)/packs" does not exist`)
	// ErrMissingSource indicates that information about the source of the pack repo was not found
	ErrMissingSource = errors.New("cannot get information about pack repo source")
	// ErrRepoDirty indicates that the pack repo was modified
	ErrRepoDirty = errors.New("pack repo was modified")
	//ErrVersionDoesNotExist indicates that the requested pack repo version does not exist
	ErrVersionDoesNotExist = errors.New("requested version does not exist")
)
