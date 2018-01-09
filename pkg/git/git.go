package git

import (
	"path/filepath"
	"github.com/jenkins-x/jx/pkg/util"
)

// FindGitConfigDir tries to find the `.git` directory either in the current directory or in parent directories
func FindGitConfigDir(dir string) (string, string, error) {
	d := dir
	for {
		gitDir := filepath.Join(d, ".git/config")
		f, err := util.FileExists(gitDir)
		if err != nil {
			return "", "", err
		}
		if f {
			return d, gitDir, nil
		}
		p, _ := filepath.Split(d)
		if d == "/" || p == d {
			return "", "", nil
		}
	}

}
