package tests

import (
	"fmt"
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
