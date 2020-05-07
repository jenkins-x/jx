package util

import "runtime"

// GetSh returns the default sh path.
// Windows returns sh, other platform returns /bin/sh.
func GetSh() string {
	shell := "/bin/sh"
	if runtime.GOOS == "windows" {
		shell = "sh"
	}

	return shell
}
