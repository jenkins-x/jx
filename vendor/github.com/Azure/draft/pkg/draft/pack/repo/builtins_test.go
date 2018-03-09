package repo

import (
	"testing"

	"github.com/Azure/draft/pkg/version"
)

const stableRelease = "v1.0.0"

func TestBuiltins(t *testing.T) {
	version.Release = "canary"

	builtins := Builtins()
	if len(builtins) != 1 {
		t.Errorf("expected 1 builtin, got %d", len(builtins))
	}

	if builtins[0].URL != "https://github.com/Azure/draft" {
		t.Error("expected https://github.com/Azure/draft to be in the builtin list")
	}

	if builtins[0].Name != "github.com/Azure/draft" {
		t.Error("expected github.com/Azure/draft to be in the builtin list")
	}

	if builtins[0].Version != "" {
		t.Error("expected version to be an empty string when the current draft release is a canary release")
	}

	version.Release = stableRelease

	builtins = Builtins()

	if builtins[0].Version != stableRelease {
		t.Errorf("expected version to be '%s'; got '%s'", stableRelease, builtins[0].Version)
	}
}
