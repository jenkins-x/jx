// +build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
)

// runHook will execute a plugin hook.
func runHook(p *plugin.Plugin, event string) error {
	hooks, ok := p.Metadata.PlatformHooks["bash"]
	if !ok {
		return nil
	}
	hook := hooks.Get(event)
	if hook == "" {
		return nil
	}

	prog := exec.Command("sh", "-c", hook)

	debug("running %s hook: %s %v", event, prog.Path, prog.Args)

	home := draftpath.Home(homePath())
	setupPluginEnv(p.Metadata.Name, p.Metadata.Version, p.Dir, home.Plugins(), home)
	prog.Stdout, prog.Stderr = os.Stdout, os.Stderr
	if err := prog.Run(); err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			os.Stderr.Write(eerr.Stderr)
			return fmt.Errorf("plugin %s hook for %q exited with error", event, p.Metadata.Name)
		}
		return err
	}
	return nil
}
