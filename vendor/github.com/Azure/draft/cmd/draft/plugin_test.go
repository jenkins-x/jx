package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

func TestManuallyProcessArgs(t *testing.T) {
	input := []string{
		"--debug",
		"--foo", "bar",
		"--host", "example.com",
		"--kube-context", "test1",
		"--home=/tmp",
		"command",
	}

	expectKnown := []string{
		"--debug", "--host", "example.com", "--kube-context", "test1", "--home=/tmp",
	}

	expectUnknown := []string{
		"--foo", "bar", "command",
	}

	known, unknown := manuallyProcessArgs(input)

	for i, k := range known {
		if k != expectKnown[i] {
			t.Errorf("expected known flag %d to be %q, got %q", i, expectKnown[i], k)
		}
	}
	for i, k := range unknown {
		if k != expectUnknown[i] {
			t.Errorf("expected unknown flag %d to be %q, got %q", i, expectUnknown[i], k)
		}
	}

}

func TestLoadPlugins(t *testing.T) {
	// Set draft home to point to testdata
	old := draftHome
	draftHome = "testdata/drafthome"
	resetEnvVars := unsetEnvVars()
	defer func() {
		draftHome = old
		resetEnvVars()
	}()
	ph := draftpath.Home(homePath())

	out := bytes.NewBuffer(nil)
	in := bytes.NewBuffer(nil)
	cmd := &cobra.Command{}
	// add `--home` flag to cmd (which is what cmd.Parent() resolves to for the loaded plugin)
	// so that it can be overridden in tests below
	p := cmd.PersistentFlags()
	p.StringVar(&draftHome, "home", draftHome, "location of your Draft config. Overrides $DRAFT_HOME")

	loadPlugins(cmd, ph, out, in)

	envs := strings.Join([]string{
		"fullenv",
		ph.Plugins() + "/fullenv",
		ph.Plugins(),
		ph.String(),
		os.Args[0],
	}, "\n")

	// testVariant represents an expect and args variant of a given test/plugin
	type testVariant struct {
		expect string
		args   []string
	}

	// Test that the YAML file was correctly converted to a command.
	tests := []struct {
		use      string
		short    string
		long     string
		variants []testVariant
	}{
		{"args", "echo args", "This echos args", []testVariant{
			{expect: "-a -b -c\n", args: []string{"-a", "-b", "-c"}},
		}},
		{"echo", "echo stuff", "This echos stuff", []testVariant{
			{expect: "hello\n", args: []string{}},
		}},
		{"fullenv", "show env vars", "show all env vars", []testVariant{
			{expect: envs + "\n", args: []string{}},
		}},
		{"home", "home stuff", "show DRAFT_HOME", []testVariant{
			{expect: ph.String() + "\n", args: []string{}},
			{expect: "/my/draft/home\n", args: []string{"--home", "/my/draft/home"}},
		}},
	}

	plugins := cmd.Commands()

	if len(plugins) != len(tests) {
		t.Fatalf("Expected %d plugins, got %d", len(tests), len(plugins))
	}

	for i := 0; i < len(plugins); i++ {
		tt := tests[i]
		pp := plugins[i]
		if pp.Use != tt.use {
			t.Errorf("%d: Expected Use=%q, got %q", i, tt.use, pp.Use)
		}
		if pp.Short != tt.short {
			t.Errorf("%d: Expected Use=%q, got %q", i, tt.short, pp.Short)
		}
		if pp.Long != tt.long {
			t.Errorf("%d: Expected Use=%q, got %q", i, tt.long, pp.Long)
		}

		for _, variant := range tt.variants {
			out.Reset()
			// Currently, plugins assume a Linux subsystem. Skip the execution
			// tests until this is fixed
			if runtime.GOOS != "windows" {
				if err := pp.RunE(pp, variant.args); err != nil {
					t.Errorf("Error running %s: %s", tt.use, err)
				}
				if out.String() != variant.expect {
					t.Errorf("Expected %s to output:\n%s\ngot\n%s", tt.use, variant.expect, out.String())
				}
			}
		}
	}
}

func TestSetupEnv(t *testing.T) {
	name := "pequod"
	ph := draftpath.Home("testdata/drafthome")
	base := filepath.Join(ph.Plugins(), name)
	plugdirs := ph.Plugins()
	flagDebug = true
	defer func() {
		flagDebug = false
	}()

	resetEnvVars := unsetEnvVars()
	defer resetEnvVars()
	setupPluginEnv(name, base, plugdirs, ph)
	for _, tt := range []struct {
		name   string
		expect string
	}{
		{"DRAFT_PLUGIN_NAME", name},
		{"DRAFT_PLUGIN_DIR", base},
		{"DRAFT_PLUGIN", ph.Plugins()},
		{"DRAFT_DEBUG", "1"},
		{"DRAFT_HOME", ph.String()},
		{"DRAFT_PACKS_HOME", ph.Packs()},
		{"HELM_HOST", tillerHost},
	} {
		if got := os.Getenv(tt.name); got != tt.expect {
			t.Errorf("Expected $%s=%q, got %q", tt.name, tt.expect, got)
		}
	}
}

func unsetEnvVars() func() {
	envs := []string{"DRAFT_PLUGIN_NAME", "DRAFT_PLUGIN_DIR", "DRAFT_PLUGIN", "DRAFT_DEBUG", "DRAFT_HOME", "DRAFT_PACKS_HOME", "DRAFT_HOST"}

	resetVals := map[string]string{}

	for _, env := range envs {
		val := os.Getenv(env)
		resetVals[env] = val
		if err := os.Unsetenv(env); err != nil {
			debug("error unsetting env %v: %v", env, err)
		}
	}

	return func() {
		for env, val := range resetVals {
			if err := os.Setenv(env, val); err != nil {
				debug("error setting env variable %s to %s: %s", env, val, err)
			}
		}
	}
}
