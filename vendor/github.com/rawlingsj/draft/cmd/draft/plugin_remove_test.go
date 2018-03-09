package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/testing/helpers"
)

func TestPluginRemoveCmd(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}
	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTestPluginEnv(target, old)

	remove := &pluginRemoveCmd{
		home:  draftpath.Home(homePath()),
		out:   buf,
		names: []string{"echo"},
	}

	helpers.CopyTree(t, "testdata/plugins/", pluginDirPath(remove.home))

	if err := remove.run(); err != nil {
		t.Errorf("Error removing plugin: %v", err)
	}

	expectedOutput := "Removed plugin: echo\n"
	actual := buf.String()

	if strings.Compare(expectedOutput, actual) != 0 {
		t.Errorf("Expected %v, got %v", expectedOutput, actual)
	}
}
