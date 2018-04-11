package repo

import (
	"testing"
)

func TestPack(t *testing.T) {
	r := Repository{
		Name: "testRepo1",
		Dir:  "testdata/packs/github.com/testOrg1/testRepo1",
	}

	targetPack := "testpack2"
	expected := "testdata/packs/github.com/testOrg1/testRepo1/packs/testpack2"
	pack, err := r.Pack(targetPack)
	if err != nil {
		t.Fatal(err)
	}
	if pack != expected {
		t.Errorf("Expected pack %s, got %s", expected, pack)
	}

}

func TestPackNotFound(t *testing.T) {
	r := Repository{
		Name: "testRepo1",
		Dir:  "testdata/packs/github.com/testOrg1/testRepo1",
	}
	targetPack := "nopack"
	if _, err := r.Pack(targetPack); err == nil {
		t.Error("Expected error, got no error")
	}
}
