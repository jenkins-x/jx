package pack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFrom(t *testing.T) {
	tdir, err := ioutil.TempDir("", "pack-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tdir)

	if err := CreateFrom(tdir, "testdata/pack-python"); err != nil {
		t.Errorf("expected err to be nil, got %v", err)
	}

	// verify that canada.txt was copied over
	if _, err := os.Stat(filepath.Join(tdir, "canada.txt")); err != nil {
		if os.IsNotExist(err) {
			t.Error("expected Canada to exist")
		}
	}

	if err := CreateFrom(tdir, "testdata/pack-does-not-exist"); err == nil {
		t.Error("expected err to be non-nil with an invalid source pack")
	}
}
