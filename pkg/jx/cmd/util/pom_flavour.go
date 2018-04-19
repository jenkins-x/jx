package util

import (
	"io/ioutil"
	"strings"
)

const (
	Liberty = "liberty"
)

func PomFlavour(path string) (string, error) {

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil
	}

	s := string(b)
	if strings.Contains(s, "<packaging>war</packaging>") &&
		strings.Contains(s, "org.eclipse.microprofile") {
		return Liberty, nil
	}

	return "", nil

}
