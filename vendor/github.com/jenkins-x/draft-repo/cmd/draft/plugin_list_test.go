package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

func TestPluginListCmd(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	resetEnvVars := unsetEnvVars()
	defer resetEnvVars()

	list := &pluginListCmd{
		home: draftpath.Home("testdata/drafthome/"),
		out:  buf,
	}

	if err := list.run(); err != nil {
		t.Errorf("draft plugin list error: %v", err)
	}

	expectedOutput := "NAME   \tVERSION\tDESCRIPTION      \nargs   \t       \tThis echos args  \necho   \t       \tThis echos stuff \nfullenv\t       \tshow all env vars\nhome   \t       \tshow DRAFT_HOME  \n"

	actual := buf.String()
	if strings.Compare(actual, expectedOutput) != 0 {
		t.Errorf("Expected %q, Got %q", expectedOutput, actual)
	}
}

func TestEmptyResultsOnPluginListCmd(t *testing.T) {
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}

	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}

	defer teardownTestPluginEnv(target, old)

	buf := bytes.NewBuffer(nil)
	list := &pluginListCmd{
		home: draftpath.Home(homePath()),
		out:  buf,
	}

	if err := list.run(); err != nil {
		t.Errorf("draft plugin list error: %v", err)
	}

	expectedOutput := "No plugins found\n"
	actual := buf.String()
	if strings.Compare(actual, expectedOutput) != 0 {
		t.Errorf("Expected %s, got %s", expectedOutput, actual)
	}

}
