// +build !linux
// +build !darwin
// +build !windows

package system

import (
	"fmt"
	"runtime"
)

// returns a human friendly string of the current OS
func GetOsVersion() (string,  error) {
	str := fmt.Sprintf("unknown platform %s %s", runtime.GOOS, runtime.GOARCH)
	return str, fmt.ErrorF(str)
}