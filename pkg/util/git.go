package util

import (
	"errors"
	"fmt"
	"strings"
)

// GetRemoteAndRepo splits a refspec string (eg, jenkins-x/jjx) into the two strings for the origin (jenkins-x)
// and repo (jx)
func GetRemoteAndRepo(refspec string) (origin, repo string, err error) {
	s := strings.Split(refspec, "/")
	if len(s) != 2 {
		err = errors.New(fmt.Sprintf("%s is not of the format org/repo", refspec))
	} else {
		origin = s[0]
		repo = s[1]
	}
	return
}
