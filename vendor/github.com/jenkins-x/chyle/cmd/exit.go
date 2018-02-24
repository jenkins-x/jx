package cmd

import (
	"os"
)

var exitError = func() {
	os.Exit(1)
}

var exitSuccess = func() {
	os.Exit(0)
}
