package main

import (
	"os"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
)

func TestInitClientOnly(t *testing.T) {
	resetEnvVars := unsetEnvVars()
	tempHome, _ := tempDir(t, "draft-init")
	os.Setenv(homeEnvVar, tempHome)
	defer func() {
		teardown()
		resetEnvVars()
	}()

	cmd := &initCmd{
		clientOnly: true,
		out:        os.Stdout,
		in:         os.Stdin,
		home:       draftpath.Home(tempHome),
	}

	if err := cmd.run(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	plugins, err := findPlugins(cmd.home.Plugins())
	if err != nil {
		t.Fatal(err)
	}

	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %v", len(plugins))
	}

	repos := repo.FindRepositories(cmd.home.Packs())
	if len(repos) != 1 {
		t.Errorf("Expected 1 pack repo, got %v", len(repos))
	}
}
