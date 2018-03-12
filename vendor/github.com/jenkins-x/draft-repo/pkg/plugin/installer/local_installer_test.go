package installer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

var _ Installer = new(LocalInstaller)

func TestLocalInstaller(t *testing.T) {
	dh, err := ioutil.TempDir("", "draft-home-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dh)

	home := draftpath.Home(dh)
	if err := os.MkdirAll(home.Plugins(), 0755); err != nil {
		t.Fatalf("Could not create %s: %s", home.Plugins(), err)
	}

	// Make a temp dir
	tdir, err := ioutil.TempDir("", "draft-installer-")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tdir)

	if err := ioutil.WriteFile(filepath.Join(tdir, "plugin.yaml"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	source := "../testdata/plugdir/echo"
	i, err := New(source, "", home)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := Install(i); err != nil {
		t.Error(err)
	}

	if i.Path() != home.Path("plugins", "echo") {
		t.Errorf("expected path '$DRAFT_HOME/plugins/draft-env', got %q", i.Path())
	}
}
