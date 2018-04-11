package pack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/chartutil"
)

// FromDir takes a string name, tries to resolve it to a file or directory, and then loads it.
//
// This is the preferred way to load a pack. It will discover the pack encoding
// and hand off to the appropriate pack reader.
func FromDir(dir string) (*Pack, error) {
	pack := new(Pack)
	pack.Files = make(map[string]io.ReadCloser)

	topdir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	pack.Chart, err = chartutil.LoadDir(filepath.Join(topdir, ChartsDir))
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(topdir)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %s", topdir, err)
	}

	// load all files in the lowest level of the pack directory
	for _, fInfo := range files {
		if !fInfo.IsDir() {
			f, err := os.Open(filepath.Join(topdir, fInfo.Name()))
			if err != nil {
				return nil, err
			}
			if fInfo.Name() != "README.md" {
				pack.Files[fInfo.Name()] = f
			}
		}
	}

	return pack, nil
}
