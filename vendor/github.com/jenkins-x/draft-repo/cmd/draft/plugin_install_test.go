package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

type pluginTest struct {
	name   string
	plugin string
	path   string
	output string
	fail   bool
	flags  []string
}

type testPluginEnv struct {
	pluginEnvVar string
	draftHome    string
}

func setupTestPluginEnv(target *testPluginEnv) (*testPluginEnv, error) {
	// save old
	old := draftHome
	oldenv := os.Getenv(pluginEnvVar)

	// set new
	draftHome = target.draftHome
	err := os.Setenv(pluginEnvVar, target.pluginEnvVar)

	return &testPluginEnv{
		draftHome:    old,
		pluginEnvVar: oldenv,
	}, err
}

func teardownTestPluginEnv(current, original *testPluginEnv) {
	draftHome = original.draftHome
	os.Setenv(pluginEnvVar, original.pluginEnvVar)
	os.RemoveAll(current.draftHome)
}

func newTestPluginEnv(home, pluginEnvVarValue string) (*testPluginEnv, error) {
	target := &testPluginEnv{}

	if home == "" {
		tempHome, err := ioutil.TempDir("", "draft_home-")
		if err != nil {
			return target, err
		}

		if err := os.Mkdir(filepath.Join(tempHome, "plugins"), 0755); err != nil {
			return target, err
		}

		target.draftHome = tempHome
	} else {
		target.draftHome = home
	}

	target.pluginEnvVar = pluginEnvVarValue

	return target, nil
}

func TestPluginInstallCmd(t *testing.T) {
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}

	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTestPluginEnv(target, old)

	tests := []pluginTest{
		{
			name:   "install plugin",
			plugin: "echo",
			path:   "testdata/plugins/echo",
			output: "Installed plugin: echo\n",
			fail:   false,
		},
		{
			name:   "error installing nonexistent plugin",
			plugin: "dummy",
			path:   "testdata/plugins/dummy",
			output: "",
			fail:   true,
		},
	}

	home := draftpath.Home(draftHome)
	buf := bytes.NewBuffer(nil)
	for _, tt := range tests {
		cmd := newPluginInstallCmd(buf)

		if err := cmd.PreRunE(cmd, []string{tt.path}); err != nil {
			t.Errorf("%q reported error: %s", tt.name, err)
		}

		if err := cmd.RunE(cmd, []string{tt.path}); err != nil && !tt.fail {
			t.Errorf("%q reported error: %s", tt.name, err)
		}

		if !tt.fail {
			result := buf.String()
			if strings.Compare(result, tt.output) != 0 {
				t.Errorf("Expected %v, got %v", tt.output, result)
			}

			if _, err = os.Stat(filepath.Join(home.Plugins(), tt.plugin)); err != nil && os.IsNotExist(err) {
				t.Errorf("Installed plugin not found: %v", err)
			}

		}

		buf.Reset()
	}

	cmd := newPluginInstallCmd(buf)
	if err := cmd.PreRunE(cmd, []string{"arg1", "extra arg"}); err == nil {
		t.Error("Expected failure due to incorrect number of arguments for plugin install command")
	}

}
