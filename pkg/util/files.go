package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultWritePermissions = 0760

	MaximumNewDirectoryAttempts = 1000
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

func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
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

func RenameDir(src string, dst string, force bool) (err error) {
	err = CopyDir(src, dst, force)
	if err != nil {
		return fmt.Errorf("failed to copy source dir %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source dir %s: %s", src, err)
	}
	return nil
}

func RenameFile(src string, dst string) (err error) {
	err = CopyFile(src, dst)
	if err != nil {
		return fmt.Errorf("failed to copy source file %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source file %s: %s", src, err)
	}
	return nil
}

// credit https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func CopyDir(src string, dst string, force bool) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		if force {
			os.RemoveAll(dst)
		} else {
			return fmt.Errorf("destination already exists")
		}
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath, force)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

// credit https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDirOverwrite copies from the source dir to the destination dir overwriting files along the way
func CopyDirOverwrite(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDirOverwrite(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}
	return
}

// loads a file
func LoadBytes(dir, name string) ([]byte, error) {
	path := filepath.Join(dir, name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading file %s in directory %s, %v", name, dir, err)
	}
	return bytes, nil
}

func DeleteFile(fileName string) (err error) {
	if fileName != "" {
		exists, err := FileExists(fileName)
		if err != nil {
			return fmt.Errorf("Could not check if file exists %s due to %s", fileName, err)
		}

		if exists {
			err = os.Remove(fileName)
			if err != nil {
				return fmt.Errorf("Could not remove file due to %s", fileName, err)
			}
		}
	} else {
		return fmt.Errorf("Filename is not valid")
	}
	return nil
}
