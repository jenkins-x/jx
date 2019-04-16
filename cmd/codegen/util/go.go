package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

const (
	gopath = "GOPATH"
)

// GoPath returns the first element of the GOPATH.
// The empty string is returned if GOPATH is not set.
func GoPath() string {
	goPath := os.Getenv(gopath)

	// GOPATH can have multiple elements, we take the first which is consistent with what 'go get' does
	pathElements := strings.Split(goPath, string(os.PathListSeparator))
	path := pathElements[0]
	return path
}

// GoPathSrc returns the src directory of the first GOPATH element.
func GoPathSrc() string {
	return filepath.Join(GoPath(), "src")
}

// GoPathBin returns the bin directory of the first GOPATH element.
func GoPathBin() string {
	return filepath.Join(GoPath(), "bin")
}

// EnsureGoPath ensures the GOPATH environment variable is set and points to a valid directory.
func EnsureGoPath() error {
	goPath := os.Getenv(gopath)
	if goPath == "" {
		return errors.New("GOPATH needs to be set")
	}

	// GOPATH can have multiple elements, if so take the first
	pathElements := strings.Split(goPath, string(os.PathListSeparator))
	path := pathElements[0]
	if len(pathElements) > 1 {
		AppLogger().Debugf("GOPATH contains more than one element using %s", path)
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("the GOPATH directory %s does not exist", path))
	} else {
		return err
	}
}

// GoGet runs go get to install the specified binary.
func GoGet(path string, version string, goModules bool) error {
	modulesMode := "off"
	if goModules {
		modulesMode = "on"
	}

	fullPath := path
	if version != "" {
		fullPath = fmt.Sprintf("%s@%s", path, version)
	}

	goGetCmd := util.Command{
		Name: "go",
		Args: []string{
			"get",
			fullPath,
		},
		Env: map[string]string{
			"GO111MODULE": modulesMode,
		},
	}
	out, err := goGetCmd.RunWithoutRetry()
	if err != nil {
		return errors.Wrapf(err, "error running %s, output %s", goGetCmd.String(), out)
	}

	return nil
}

// GetModuleRequirements returns the requirements for the GO module rooted in dir
// It returns a map[<module name>]map[<requirement name>]<requirement version>
func GetModuleRequirements(dir string) (map[string]map[string]string, error) {
	cmd := util.Command{
		Dir:  dir,
		Name: "go",
		Args: []string{
			"mod",
			"graph",
		},
		Env: map[string]string{
			"GO111MODULE": "on",
		},
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return nil, errors.Wrapf(err, "running %s, output %s", cmd.String(), out)
	}
	answer := make(map[string]map[string]string)
	// deal with windows
	out = strings.Replace(out, "\r\n", "\n", -1)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			return nil, errors.Errorf("line of go mod graph should be like '<module> <requirement>' but was %s",
				line)
		}
		moduleName := parts[0]
		requirement := parts[1]
		parts1 := strings.Split(requirement, "@")
		if len(parts1) != 2 {
			return nil, errors.Errorf("go mod graph line should be like '<module> <requirementName>@<requirementVersion>' but was %s", line)
		}
		requirementName := parts1[0]
		requirementVersion := parts1[1]
		if _, ok := answer[moduleName]; !ok {
			answer[moduleName] = make(map[string]string)
		}
		answer[moduleName][requirementName] = requirementVersion
	}
	return answer, nil
}
