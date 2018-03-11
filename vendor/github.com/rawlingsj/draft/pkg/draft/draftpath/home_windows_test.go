package draftpath

import (
	"testing"
)

func TestDraftHome(t *testing.T) {
	ph := Home("r:\\")
	isEq := func(t *testing.T, a, b string) {
		if a != b {
			t.Errorf("Expected %q, got %q", b, a)
		}
	}

	isEq(t, ph.String(), "r:\\")
	isEq(t, ph.Packs(), "r:\\packs")
	isEq(t, ph.Plugins(), "r:\\plugins")
}
