package ksync

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/v2/pkg/packages"
	"github.com/jenkins-x/jx/v2/pkg/util"
)

var commit string
var binary = "ksync"

// CLI implements the command ksync commands contained in KSyncer.
type CLI struct {
	Runner util.Commander
}

// NewCLI creates a new KsyncCLI instance configured to use the provided ksync CLI in the given current working directory
func NewCLI() (*CLI, error) {
	path, err := util.JXBinLocation()
	if err != nil {
		return nil, err
	}
	runner := &util.Command{
		Name: fmt.Sprintf("%s/ksync", path),
	}
	cli := &CLI{
		Runner: runner,
	}
	return cli, nil
}

// isInstalled checks if the binary is installed
func isInstalled(binary string) (bool, error) {
	flag, err := packages.ShouldInstallBinary(binary)
	return flag, err
}

// getTag returns the latest full tag from github given the owner and repo, and installs
func getTag(owner, repo string) (string, string, error) {
	latestTag, err := util.GetLatestFullTagFromGithub(owner, repo)
	if err != nil {
		return "", "", err
	}
	latestVersion := *latestTag.Name
	// This will return the shortsha, the first 7 characters associated with the git commit
	latestSha := (*latestTag.Commit.SHA)[:7]
	return latestSha, latestVersion, nil
}

// install downloads and installs the ksync package
func install(latestVersion, binDir, fileName string) error {
	clientURL := fmt.Sprintf("https://github.com/ksync/ksync/releases/download/%s/ksync_%s_%s", latestVersion, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		clientURL += ".exe"
	}
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err := packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	err = os.Chmod(fullPath, 0755)
	if err != nil {
		return err
	}
	return nil
}

// Version prints out the version of the local component of ksync
func (kcli *CLI) Version() (string, error) {
	args := []string{"version"}
	kcli.Runner.SetArgs(args)
	out, _ := kcli.Runner.RunWithoutRetry()
	return out, nil
}

// Init installs the server side component of ksync - the daemonsets
func (kcli *CLI) Init(flags ...string) (string, error) {
	args := []string{"init"}
	args = append(args, flags...)
	kcli.Runner.SetArgs(args)
	out, err := kcli.Runner.RunWithoutRetry()
	if err != nil {
		return "", err
	}
	return out, nil
}

// Clean removes the ksync pods
func (kcli *CLI) Clean() (string, error) {
	args := []string{"clean"}
	kcli.Runner.SetArgs(args)
	out, err := kcli.Runner.RunWithoutRetry()
	if err != nil {
		return "", err
	}
	return out, nil
}

// InstallKSync installs ksync, it returns the sha of the latest commit
func InstallKSync() (string, error) {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return "", err
	}

	download, err := isInstalled(binary)
	if err != nil || !download {
		// Exec `ksync` to find the version

		kcli, err := NewCLI()
		if err != nil {
			return "", err
		}
		res, err := kcli.Version()
		if err != nil {
			return "", err
		}

		commit = getCommitShaFromVersion(res)
		if commit != "" {
			return commit, nil
		}
		return "", fmt.Errorf("unable to find version of ksync")
	}
	latestSha, latestVersion, err := getTag("ksync", "ksync")
	if err != nil {
		return "", err
	}

	err = install(latestVersion, binDir, packages.BinaryWithExtension(binary))
	if err != nil {
		return "", err
	}

	return latestSha, nil
}

func getCommitShaFromVersion(result string) string {
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Git Commit:") {
			commit = strings.TrimSpace(strings.TrimPrefix(line, "Git Commit:"))
		}
	}
	return commit
}
