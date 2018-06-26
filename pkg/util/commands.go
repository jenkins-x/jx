package util

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func PathWithBinary() string {
	path := os.Getenv("PATH")
	binDir, _ := BinaryLocation()
	answer := path + string(os.PathListSeparator) + binDir
	mvnBinDir, _ := MavenBinaryLocation()
	if mvnBinDir != "" {
		answer += string(os.PathListSeparator) + mvnBinDir
	}
	return answer
}

// RunCommandWithOutput evaluates the given command and returns the trimmed output
func RunCommandWithOutput(dir string, name string, args ...string) (string, error) {
	os.Setenv("PATH", PathWithBinary())
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	if err != nil {
		return text, errors.Wrapf(err, "failed to run '%s %s' command in directory '%s', output: '%s'",
			name, strings.Join(args, " "), dir, text)
	}
	return text, err
}

// RunCommand evaluates the given command and returns the trimmed output
func RunCommand(dir string, name string, args ...string) error {
	os.Setenv("PATH", PathWithBinary())
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = os.Stdout
	e.Stderr = os.Stdin
	err := e.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to run '%s %s' command in directory '%s'", name, strings.Join(args, " "), dir)
	}
	return err
}
