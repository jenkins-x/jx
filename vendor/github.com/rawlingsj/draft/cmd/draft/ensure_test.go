package main

import (
	"os"
	"testing"

	pluginbase "k8s.io/helm/pkg/plugin"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
)

func TestEnsureDirectories(t *testing.T) {
	resetEnvVars := unsetEnvVars()
	tempHome, teardown := tempDir(t, "draft-init")
	defer func() {
		teardown()
		resetEnvVars()
	}()

	cmd := &initCmd{
		home: draftpath.Home(tempHome),
		out:  os.Stdout,
	}

	if err := cmd.ensureDirectories(); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(cmd.home.String())
	if err != nil {
		t.Errorf("Expected home directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected home to be directory but isn't")
	}

	fi, err = os.Stat(cmd.home.Plugins())
	if err != nil {
		t.Errorf("Expected plugins directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected plugins to be directory but isn't")
	}

	fi, err = os.Stat(cmd.home.Packs())
	if err != nil {
		t.Errorf("Expected packs directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected packs to be directory but isn't")
	}

}

func TestEnsurePlugin(t *testing.T) {
	resetEnvVars := unsetEnvVars()
	tempHome, teardown := tempDir(t, "draft-init")
	defer func() {
		teardown()
		resetEnvVars()
	}()

	cmd := &initCmd{
		home: draftpath.Home(tempHome),
		out:  os.Stdout,
		in:   os.Stdin,
	}

	if err := os.MkdirAll(cmd.home.Plugins(), 0755); err != nil {
		t.Fatalf("Could not create %s: %s", cmd.home.Plugins(), err)
	}

	builtinPlugin := &plugin.Builtin{Name: "echo", Version: "1.0.0", URL: "testdata/plugins/echo"}
	empty := []*pluginbase.Plugin{}

	if err := cmd.ensurePlugin(builtinPlugin, empty); err != nil {
		t.Fatal(err)
	}
	installed, err := findPlugins(pluginDirPath(cmd.home))
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Errorf("Expected 1 plugin to be installed, got %v", len(installed))
	}
}

func TestEnsurePluginExisting(t *testing.T) {
	resetEnvVars := unsetEnvVars()
	tempHome, teardown := tempDir(t, "draft-init")
	defer func() {
		teardown()
		resetEnvVars()
	}()

	cmd := &initCmd{
		home: draftpath.Home(tempHome),
		out:  os.Stdout,
		in:   os.Stdin,
	}

	builtinPlugin := &plugin.Builtin{Name: "something", Version: "1.0.0"}
	existingPlugins := []*pluginbase.Plugin{
		&pluginbase.Plugin{Metadata: &pluginbase.Metadata{
			Name: "something", Version: "1.0.0"},
		},
	}
	if err := cmd.ensurePlugin(builtinPlugin, existingPlugins); err != nil {
		t.Fatal(err)
	}

}
