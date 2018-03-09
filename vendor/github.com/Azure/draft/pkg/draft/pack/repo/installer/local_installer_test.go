package installer

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin/installer"
)

var _ installer.Installer = new(LocalInstaller)

func TestLocalInstaller(t *testing.T) {
	dh, err := ioutil.TempDir("", "draft-home-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dh)

	home := draftpath.Home(dh)
	if err := os.MkdirAll(home.Packs(), 0755); err != nil {
		t.Fatalf("Could not create %s: %s", home.Packs(), err)
	}

	// Make a temp dir
	tdir, err := ioutil.TempDir("", "draft-installer-")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tdir)

	source := "testdata/packdir/defaultpacks"
	i, err := New(source, "", home)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := Install(i); err != nil {
		t.Error(err)
	}

	if i.Path() != home.Path("packs", "defaultpacks") {
		t.Errorf("expected path '$DRAFT_HOME/packs/defaultpacks', got %q", i.Path())
	}
}
