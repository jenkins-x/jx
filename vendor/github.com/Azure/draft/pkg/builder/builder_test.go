package builder

import (
	"testing"

	"github.com/Azure/draft/pkg/draft/manifest"
)

func TestArchiveSrc(t *testing.T) {
	ctx := &Context{
		AppDir: "testdata/simple",
		Env: &manifest.Environment{
			Dockerfile: "",
		},
	}

	if err := archiveSrc(ctx); err != nil {
		t.Error(err)
	}

	if ctx.SrcName != "build.tar.gz" {
		t.Errorf("expected %s, got %s", "build.tar.gz", ctx.AppDir)
	}
	if len(ctx.Archive) == 0 {
		t.Errorf("expected non-zero archive length, got %d", len(ctx.Archive))
	}
}
