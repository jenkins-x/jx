package builds

import (
	"regexp"
	"strings"
)

var (
	numericStringRegex = regexp.MustCompile("[0-9]+")
)
// GetBuildNumberFromLabelsFileData parses the /etc/podinfo/labels style downward API file for a pods labels
// and returns the build number if it can be discovered
func GetBuildNumberFromLabelsFileData(text string) string {
   m := LoadDownwardAPILabels(text)
   answer := m["build.knative.dev/buildName"]
   if answer == "" {
	   answer = m["build-number"]
   }
   if answer != "" {
   		return lastNumberFrom(answer)
   }
   return ""
}

// lastNumberFrom splits a string such as "jstrachan-mynodething-master-1-build" via "-" and returns the last
// numeric string
func lastNumberFrom(text string) string {
	// lets remove any whilespace or double quotes
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "\"")
	text = strings.TrimSuffix(text, "\"")

	paths := strings.Split(text, "-")
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		if numericStringRegex.MatchString(path) {
			return path
		}
	}
	return ""
}

// LoadDownwardAPILabels parses the /etc/podinfo/labels text into a map of label values
func LoadDownwardAPILabels(text string) map[string]string {
	m := map[string]string{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		l := strings.TrimSpace(line)
		paths := strings.SplitN(l, "=", 2)
		if len(paths) == 2 {
			m[paths[0]] = paths[1]
		}
	}
	return m
}
