package util

import (
	"os"
	"path/filepath"
	"strconv"
	"fmt"
)

const (
	DefaultWritePermissions = 0760
)
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}


// CreateUniqueDirectory creates a new directory but if the combination of dir and name exists
// then append a number until a unique name is found
func CreateUniqueDirectory(dir string, name string, maximumAttempts int) (string, error) {
	for i := 0; i < maximumAttempts; i++ {
		n := name
		if i > 0 {
			n += strconv.Itoa(i)
		}
		p := filepath.Join(dir, n)
		exists, err := FileExists(p)
		if err != nil {
			return p, err
		}
		if !exists {
			err := os.MkdirAll(p, DefaultWritePermissions)
			if err != nil {
				return "", fmt.Errorf("Failed to create directory %s due to %s", p, err)
			}
			return p, nil
		}
	}
	return "", fmt.Errorf("Could not create a unique file in %s starting with %s after %d attempts", dir, name, maximumAttempts)
}
