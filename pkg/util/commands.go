package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func PathWithBinary() string {
	path := os.Getenv("PATH")
	binDir, _ := BinaryLocation()
	return path + string(os.PathListSeparator) + binDir
}

// GetCommandOutput evaluates the given command and returns the trimmed output
func GetCommandOutput(dir string, name string, args ...string) (string, error) {
	os.Setenv("PATH", PathWithBinary())
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	if err != nil {
		return text, fmt.Errorf("Error: Command failed  %s %s %s %s\n", name, strings.Join(args, " "), text, err)
	}
	return text, err
}

// RunCommand evaluates the given command and returns the trimmed output
func RunCommand(dir string, name string, args ...string) error {
	e := exec.Command(name, args...)
	if dir != "" {
		e.Dir = dir
	}
	e.Stdout = os.Stdout
	e.Stderr = os.Stdin
	err := e.Run()
	if err != nil {
		fmt.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err

}
