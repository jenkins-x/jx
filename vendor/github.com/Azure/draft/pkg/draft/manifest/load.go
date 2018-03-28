package manifest

import (
	"github.com/BurntSushi/toml"
)

// Load opens the named file for reading. If successful, the manifest is returned.
func Load(name string) (*Manifest, error) {
	mfst := New()
	if _, err := toml.DecodeFile(name, mfst); err != nil {
		return nil, err
	}
	return mfst, nil
}
