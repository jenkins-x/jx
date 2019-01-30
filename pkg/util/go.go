package util

import (
	"strings"

	"github.com/pkg/errors"
)

// GetModuleRequirements returns the requirements for the GO module rooted in dir
// It returns a map[<module name>]map[<requirement name>]<requirement version>
func GetModuleRequirements(dir string) (map[string]map[string]string, error) {
	cmd := Command{
		Dir:  dir,
		Name: "go",
		Args: []string{
			"mod",
			"graph",
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
			return nil, errors.Errorf("go mod graph line should be like '<module> <requirementName"+
				">@<requirementVersion>' but was %s", line)
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
