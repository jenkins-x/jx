package osutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExists(t *testing.T) {
	file, err := ioutil.TempFile("", "osutil")
	if err != nil {
		t.Fatal(err)
	}

	exists, err := Exists(file.Name())
	if err != nil {
		t.Errorf("expected no error when calling Exists() on a file that exists, got %v", err)
	}
	if !exists {
		t.Error("expected tempfile to exist")
	}
	os.Remove(file.Name())
	exists, err = Exists(file.Name())
	if err != nil {
		t.Errorf("expected no error when calling Exists() on a file that does not exist, got %v", err)
	}
	if exists {
		t.Error("expected tempfile to NOT exist")
	}
}

func TestSymlinkWithFallback(t *testing.T) {
	const (
		oldFileName = "foo.txt"
		newFileName = "bar.txt"
	)
	tmpDir, err := ioutil.TempDir("", "osutil")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldFileNamePath := filepath.Join(tmpDir, oldFileName)
	newFileNamePath := filepath.Join(tmpDir, newFileName)

	oldFile, err := os.Create(filepath.Join(tmpDir, oldFileName))
	if err != nil {
		t.Fatal(err)
	}
	oldFile.Close()

	if err := SymlinkWithFallback(oldFileNamePath, newFileNamePath); err != nil {
		t.Errorf("expected no error when calling SymlinkWithFallback() on a file that exists, got %v", err)
	}
	if runtime.GOOS == "windows" {
		exists, err := Exists(oldFileNamePath)
		if err != nil {
			t.Errorf("expected no error when calling Exists() on a file that does not exist, got %v", err)
		}
		if exists {
			// check that newFileName is a symlink. If this succeeds, then we are running this test as a
			// user that has permission to create symbolic links, so the old file should still exist.
			newFile, err := os.Lstat(newFileNamePath)
			if err != nil {
				t.Error(err)
			}
			if newFile.Mode() != os.ModeSymlink {
				t.Errorf("expected %s to be removed when %s is not a symbolic link", oldFileNamePath, newFileNamePath)
			}
		}
	}
}
