// +build linux

package system

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

// GetOsVersion returns a human friendly string of the current OS
// in the case of an error this still returns a valid string for the details that can be found.
func GetOsVersion() (string,  error) {
	// generic LSB compliant Linux
	output, err := runCommand("lsb_release", "-d", "-s")
	if err == nil {
		return output, nil
	}
	// redHat and co these have the OS and version
	output, err = getFileContents("/etc/redhat-release")
	if err == nil {
		return output , nil
	}
	// Debian and derivatives this is just the base (Sid) which is not quite enough to say exactly
	output, err = getFileContents("/etc/debian_version")
	if err == nil {
		return fmt.Sprintf("Debian %s based distribution", output), nil
	}
	// Alpine Linux
	output, err = getFileContents("/etc/alpine-release")
	if err == nil {
		return fmt.Sprintf("Alpine Linux %s", output), nil
	}
	// procfs will tell us the kernel version
	output, err = getFileContents("/proc/version")
	if err == nil {
		return fmt.Sprintf("Unkown Linux distribution %s", output), nil
	}
	return "Unknown Linux version", fmt.Errorf("Unknown Linux version")
}

func runCommand(command string, args ...string) (string, error) {
	e := exec.Command(command, args...)
	data, err := e.CombinedOutput()
	text := string(data)
	text = strings.TrimSpace(text)
	if err != nil {
		return "", fmt.Errorf("command failed '%s %s': %s %s\n", command, strings.Join(args, " "), text, err)
	}
	return text, err
}

func getFileContents(file string) (string, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(bytes), "\n"), nil
}