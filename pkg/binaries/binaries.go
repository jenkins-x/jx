package binaries

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"os/exec"
	"runtime"

	"github.com/pkg/errors"
)

const EksctlVersion = "0.1.18"
const IBMCloudVersion = "0.10.1"
const HeptioAuthenticatorAwsVersion = "1.10.3"

func BinaryWithExtension(binary string) string {
	if runtime.GOOS == "windows" {
		if binary == "gcloud" {
			return binary + ".cmd"
		}
		return binary + ".exe"
	}
	return binary
}

func LookupForBinary(binary string) (string, error) {
	path, err := exec.LookPath(BinaryWithExtension(binary))
	if err != nil {
		return "", err
	}

	return path, nil
}

type VersionExtractor interface {
	arguments() []string

	extractVersion(command string, arguments []string) (string, error)
}

func ShouldInstallBinary(binary string, expectedVersion string, versionExtractor VersionExtractor) (bool, error) {
	if versionExtractor == nil {
		return true, nil
	}

	binaryPath, err := LookupForBinary(binary)
	if err != nil {
		return true, errors.Wrap(err, "looking up the binary")
	}
	if binaryPath != "" {
		currentVersion, err := versionExtractor.extractVersion(binaryPath, versionExtractor.arguments())
		if err != nil {
			return true, errors.Wrap(err, "extracting the version")
		}
		if currentVersion == expectedVersion {
			return false, nil
		}

	}
	return true, nil
}

func DownloadFile(clientURL string, fullPath string) error {
	log.Infof("Downloading %s to %s...\n", util.ColorInfo(clientURL), util.ColorInfo(fullPath))
	err := util.DownloadFile(fullPath, clientURL)
	if err != nil {
		return fmt.Errorf("Unable to download file %s from %s due to: %v", fullPath, clientURL, err)
	}
	log.Infof("Downloaded %s\n", util.ColorInfo(fullPath))
	return nil
}
