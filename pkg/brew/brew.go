package brew

import (
	"fmt"
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
	brewNameRegex = regexp.MustCompile(`^*.rb$`)
	//										 $1			 $2	 $3	$4	  $5	  $6	   $7
	versionRegex = regexp.MustCompile(`^(.*version) (\")((\d+\.)?(\d+\.)?(\*|\d+))(\")$`)
	//									 $1			$2	$3		   $4
	shaRegex = regexp.MustCompile(`^(.*sha256) (\")([0-9a-z]+)(\")$`)
)

//UpdateVersion scans the directory structure rooted in dir for files that match brewNameRegex and replaces any
// lines starting with FROM <name>:, ENV <name> or ARG=<name> with the newVersion
func UpdateVersion(dir string, newVersion string) ([]string, error) {
	//shaPrefix := fmt.Sprintf("url %s:", name)
	oldVersions := make(map[string]bool)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Logger().Warnf("looking for homebrew ruby files in %s %v", path, err)
		} else {
			if brewNameRegex.MatchString(filepath.Base(path)) {
				bytes, err := ioutil.ReadFile(path)
				if err != nil {
					return errors.Wrapf(err, "reading %s", path)
				}
				brewFile := string(bytes)
				answer := make([]string, 0)
				for _, line := range strings.Split(brewFile, "\n") {
					found := false
						if versionRegex.MatchString(line) {
							oldVersions[versionRegex.ReplaceAllString(line, "$3")] = true
							answer = append(answer, versionRegex.ReplaceAllString(line, fmt.Sprintf("$1 ${2}%s${7}", newVersion)))
							found = true
							continue
						}
					if !found {
						answer = append(answer, line)
					}
				}
				err = ioutil.WriteFile(path, []byte(strings.Join(answer, "\n")), info.Mode())
				if err != nil {
					return errors.Wrapf(err, "writing %s", path)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walking %s to homebrew ruby files", dir)
	}
	answer := make([]string, 0)
	for oldVersion := range oldVersions {
		answer = append(answer, oldVersion)
	}
	sort.Strings(answer)
	return answer, nil
}

//UpdateSha scans the directory structure rooted in dir for files that match brewNameRegex and replaces any
// lines starting with sha with the newVersion
func UpdateSha(dir string, newSha string) ([]string, error) {
	oldShas := make(map[string]bool)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Logger().Warnf("looking for homebrew ruby files in %s %v", path, err)
		} else {
			if brewNameRegex.MatchString(filepath.Base(path)) {
				bytes, err := ioutil.ReadFile(path)
				if err != nil {
					return errors.Wrapf(err, "reading %s", path)
				}
				brewFile := string(bytes)
				answer := make([]string, 0)
				for _, line := range strings.Split(brewFile, "\n") {
					found := false
					if shaRegex.MatchString(line) {
						oldShas[shaRegex.ReplaceAllString(line, "$3")] = true
						answer = append(answer, shaRegex.ReplaceAllString(line, fmt.Sprintf("${1} ${2}%s${4}", newSha)))
						found = true
						continue
					}
					if !found {
						answer = append(answer, line)
					}
				}
				err = ioutil.WriteFile(path, []byte(strings.Join(answer, "\n")), info.Mode())
				if err != nil {
					return errors.Wrapf(err, "writing %s", path)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walking %s to homebrew ruby files", dir)
	}
	answer := make([]string, 0)
	for oldSha := range oldShas {
		answer = append(answer, oldSha)
	}
	sort.Strings(answer)
	return answer, nil
}
