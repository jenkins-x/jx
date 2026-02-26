package ksync

// KSyncer contains the ksync commands
type KSyncer interface {
	Version() (string, error)
	Init(...string) (string, error)
	Clean() (string, error)
}
