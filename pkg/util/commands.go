package util

import (
	"os/exec"
	"strings"
)

// GetCommandOutput evaluates the given command and returns the trimmed output
func GetCommandOutput(dir string, name string, args ...string) (string, error) {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	return text, err
}

