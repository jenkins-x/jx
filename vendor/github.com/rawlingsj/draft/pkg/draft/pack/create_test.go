package pack

import (
	"io/ioutil"
	"os"
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

	if err := CreateFrom(tdir, "testdata/pack-does-not-exist"); err == nil {
		t.Error("expected err to be non-nil with an invalid source pack")
	}
}
