package brew

import (
	"github.com/jenkins-x/jx/pkg/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/pkg/errors"
)

var (
	brewFilePattern = "**/*.rb"
	versionRegex = regexp.MustCompile(`\s*version \"(.*)\"`)
	shaRegex = regexp.MustCompile(`\s*sha256 \"(.*)\"`)
)

//UpdateVersion scans the directory structure rooted in dir for files that match brewNameRegex and replaces any
// lines starting with FROM <name>:, ENV <name> or ARG=<name> with the newVersion
func UpdateVersionAndSha(dir string, newVersion string, newSha string) ([]string, []string, error) {
	oldVersions := make(map[string]bool)
	oldShas := make(map[string]bool)
	files, err := filepath.Glob(filepath.Join(dir, brewFilePattern))
	if err != nil {
		log.Logger().Warnf("looking for homebrew ruby files in %s %v", dir, err)
	}
	for _,path := range files {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "reading %s", path)
		}
		brewFile := string(bytes)
		answer := make([]string, 0)
		for _, line := range strings.Split(brewFile, "\n") {
			foundVersion, foundSha := false,false
			if versionRegex.MatchString(line) {
				answer = append(answer, util.ReplaceAllStringSubmatchFunc(versionRegex, line, func(groups []util.Group) []string {
					answer := make([]string, 0)
					for _, group := range groups {
					  oldVersions[group.Value] = true
					}
					answer = append(answer, newVersion)
					return answer
				}))
				foundVersion = true
				continue
			}
			if shaRegex.MatchString(line) {
				answer = append(answer, util.ReplaceAllStringSubmatchFunc(shaRegex, line, func(groups []util.Group) []string {
					answer := make([]string, 0)
					for _, group := range groups {
						oldShas[group.Value] = true
					}
					answer = append(answer, newSha)
					return answer
				}))
				foundSha = true
				continue
			}

			if !foundVersion || !foundSha {
				answer = append(answer, line)
			}
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "reading mode of %s file", path)
		}
		err = ioutil.WriteFile(path, []byte(strings.Join(answer, "\n")), info.Mode())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "writing %s", path)
		}
	}
	versionAnswer := make([]string, 0)
	for oldVersion := range oldVersions {
		versionAnswer = append(versionAnswer, oldVersion)
	}
	sort.Strings(versionAnswer)

	shaAnswer := make([]string, 0)
	for oldSha := range oldShas {
		shaAnswer = append(shaAnswer, oldSha)
	}
	sort.Strings(shaAnswer)
	return versionAnswer, shaAnswer, nil
}
