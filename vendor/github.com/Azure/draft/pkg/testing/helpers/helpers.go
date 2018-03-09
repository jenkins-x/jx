package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CopyTree copies src directory content tree to dest.
// If dest exists, it's deleted.
// We don't handle symlinks (not needed in this test helper)
func CopyTree(t *testing.T, src, dest string) {
	if err := os.RemoveAll(dest); err != nil {
		t.Fatalf("couldn't remove directory %s: %v", src, err)
	}

	if err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		dest := filepath.Join(dest, strings.TrimPrefix(p, src))
		if info.IsDir() {
			if err := os.MkdirAll(dest, info.Mode()); err != nil {
				return err
			}
		} else {
			data, err := ioutil.ReadFile(p)
			if err != nil {
				return err
			}
			if err = ioutil.WriteFile(dest, data, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("couldn't copy %s: %v", src, err)
	}
}
