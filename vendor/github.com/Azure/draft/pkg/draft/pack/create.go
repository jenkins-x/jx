package pack

import (
	"fmt"
)

// CreateFrom scaffolds a directory with the src pack.
func CreateFrom(dest, src string) error {
	pack, err := FromDir(src)
	if err != nil {
		return fmt.Errorf("could not load pack: %s\nTry running:\n\t$ draft pack-repo update", err)
	}
	return pack.SaveDir(dest)
}
