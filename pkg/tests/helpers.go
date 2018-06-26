package tests

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

func IsDebugLog() bool {
	return strings.ToLower(os.Getenv("JX_TEST_DEBUG")) == "true"
}

func Debugf(message string, args ...interface{}) {
	if IsDebugLog() {
		log.Infof(message, args...)
	}
}

// Output returns the output to use for tests
func Output() io.Writer {
	if IsDebugLog() {
		return os.Stdout
	}
	return ioutil.Discard
}

func TestShouldDisableMaven() bool {
	_, err := util.RunCommandWithOutput("", "mvn", "-v")
	return err != nil
}
