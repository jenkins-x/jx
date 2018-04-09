package tests

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func IsDebugLog() bool {
	return strings.ToLower(os.Getenv("JX_TEST_DEBUG")) == "true"
}

func Debugf(message string, args ...interface{}) {
	if IsDebugLog() {
		fmt.Printf(message, args...)
	}
}

// Output returns the output to use for tests
func Output() io.Writer {
	if IsDebugLog() {
		return os.Stdout
	}
	return ioutil.Discard
}
