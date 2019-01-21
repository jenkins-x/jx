package commoncmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// TODO Refactor to use util.Run or util.RunWithoutRetry?

func (o *CommonOptions) RunCommandFromDir(dir, name string, args ...string) error {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = o.Out
	e.Stderr = o.Err
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

// RunCommand runs a command
func (o *CommonOptions) RunCommand(name string, args ...string) error {
	e := exec.Command(name, args...)
	if o.Verbose {
		e.Stdout = o.Out
		e.Stderr = o.Err
	}
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) RunCommandVerbose(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) RunCommandVerboseAt(dir string, name string, args ...string) error {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = o.Out
	e.Stderr = o.Err
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) RunCommandQuietly(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = ioutil.Discard
	e.Stderr = ioutil.Discard
	os.Setenv("PATH", util.PathWithBinary())
	return e.Run()
}

func (o *CommonOptions) RunCommandInteractive(interactive bool, name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	if interactive {
		e.Stdin = os.Stdin
	}
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) RunCommandInteractiveInDir(interactive bool, dir string, name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	if interactive {
		e.Stdin = os.Stdin
	}
	if dir != "" {
		e.Dir = dir
	}
	os.Setenv("PATH", util.PathWithBinary())
	err := e.Run()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

// GetCommandOutput evaluates the given command and returns the trimmed output
func (o *CommonOptions) GetCommandOutput(dir string, name string, args ...string) (string, error) {
	os.Setenv("PATH", util.PathWithBinary())
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	if err != nil {
		return "", fmt.Errorf("Command failed '%s %s': %s %s\n", name, strings.Join(args, " "), text, err)
	}
	return text, err
}
