package osutil

import (
	"os"
	"runtime"
	"syscall"
)

// Exists returns whether the given file or directory exists or not.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// SymlinkWithFallback attempts to symlink a file or directory, but falls back to a move operation
// in the event of the user not having the required privileges to create the symlink.
func SymlinkWithFallback(oldname, newname string) (err error) {
	err = os.Symlink(oldname, newname)
	if runtime.GOOS == "windows" {
		// If creating the symlink fails on Windows because the user
		// does not have the required privileges, ignore the error and
		// fall back to renaming the file.
		//
		// ERROR_PRIVILEGE_NOT_HELD is 0x522:
		// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
		if lerr, ok := err.(*os.LinkError); ok && lerr.Err == syscall.Errno(0x522) {
			err = os.Rename(oldname, newname)
		}
	}
	return
}
