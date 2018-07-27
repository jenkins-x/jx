package util

import (
	"io/ioutil"
	"strings"
)

const (
	APPSERVER = "appserver"
	LIBERTY   = "liberty"
)

func PomFlavour(path string) (string, error) {

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil
	}

	s := string(b)
	if strings.Contains(s, "<packaging>war</packaging>") &&
		strings.Contains(s, "org.eclipse.microprofile") {
		return LIBERTY, nil
	}
	if strings.Contains(s, "<groupId>org.apache.tomcat") {
		return APPSERVER, nil
	}

	return "", nil

}
