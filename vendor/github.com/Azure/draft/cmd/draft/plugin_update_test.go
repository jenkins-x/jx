package main

import (
	"bytes"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

func TestPluginUpdateCmd(t *testing.T) {
	// move this to e2e test suite soon
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}
	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTestPluginEnv(target, old)

	home := draftpath.Home(draftHome)
	buf := bytes.NewBuffer(nil)

	update := &pluginUpdateCmd{
		names: []string{"server"},
		home:  home,
		out:   buf,
	}

	if err := update.run(); err == nil {
		t.Errorf("expected plugin update to err but did not")
	}

	install := &pluginInstallCmd{
		source:  "https://github.com/michelleN/draft-server",
		version: "0.1.0",
		home:    home,
		out:     buf,
	}

	if err := install.run(); err != nil {
		t.Fatalf("Erroring installing plugin")
	}

	if err := update.run(); err != nil {
		t.Errorf("Erroring updating plugin: %v", err)
	}

}
