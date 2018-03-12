package version

import (
	"testing"
)

func cleanUp() {
	Release = ""
	BuildMetadata = ""
	GitCommit = ""
	GitTreeState = ""
}

func TestNew(t *testing.T) {
	defer cleanUp()
	Release = "foo"
	v := New()
	if v.SemVer != "foo" {
		t.Errorf("expected 'foo', got '%s'", v.SemVer)
	}
	BuildMetadata = "bar"
	GitCommit = "car"
	GitTreeState = "star"
	v = New()
	if v.SemVer != "foo+bar" {
		t.Errorf("expected 'foo+bar', got '%s'", v.SemVer)
	}
	if v.GitCommit != "car" {
		t.Errorf("expected 'car', got '%s'", v.GitCommit)
	}
	if v.GitTreeState != "star" {
		t.Errorf("expected 'star', got '%s'", v.GitTreeState)
	}
}

func TestString(t *testing.T) {
	defer cleanUp()
	Release = "foo"
	v := New()
	if v.String() != "foo" {
		t.Errorf("expected 'foo', got '%s'", v.String())
	}
	BuildMetadata = "bar"
	v = New()
	if v.String() != "foo+bar" {
		t.Errorf("expected 'foo+bar', got '%s'", v.String())
	}
}
