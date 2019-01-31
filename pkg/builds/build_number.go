package builds

import (
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var (
	numericStringRegex = regexp.MustCompile("[0-9]+")
)


// GetBuildNumber returns the build number using environment variables and/or pod Downward API files
func GetBuildNumber() string {
	buildNumber := os.Getenv("JX_BUILD_NUMBER")
	if buildNumber != "" {
		return buildNumber
	}
	buildNumber = os.Getenv("BUILD_NUMBER")
	if buildNumber != "" {
		return buildNumber
	}
	buildID := os.Getenv("BUILD_ID")
	if buildID != "" {
		return buildID
	}
	// if we are in a knative build pod we can discover it via the dowmward API if the `/etc/podinfo/labels` file exists
	const podInfoLabelsFile = "/etc/podinfo/labels"
	exists, err := util.FileExists(podInfoLabelsFile)
	if err != nil {
	  log.Warnf("failed to detect if the file %s exists: %s\n", podInfoLabelsFile, err)
	} else if exists {
		data, err := ioutil.ReadFile(podInfoLabelsFile)
		if err != nil {
		  log.Warnf("failed to load downward API pod labels from %s due to: %s\n", podInfoLabelsFile, err)
		} else {
			text := strings.TrimSpace(string(data))
			if text != "" {
				return GetBuildNumberFromLabelsFileData(text)
			}
		}
	}
	return ""
}

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
