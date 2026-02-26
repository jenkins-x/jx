package testutils

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"

	"github.com/jenkins-x/jx/v2/pkg/cloud/amazon/session"
)

// SwitchAWSHome creates a dummy .aws dir for testing
func SwitchAWSHome() (string, error) {
	oldHome := session.UserHomeDir()
	newHome, err := ioutil.TempDir("", "common_test")
	SetUserHomeDir(newHome)
	awsHome := path.Join(newHome, ".aws")
	err = os.MkdirAll(awsHome, 0777)
	if err != nil {
		return oldHome, err
	}

	awsConfigPath := path.Join(awsHome, "config")
	if err := ioutil.WriteFile(awsConfigPath, []byte(`[profile foo]
region = bar
[profile baz]
region = qux`), 0600); err != nil {
		panic(err)
	}

	return oldHome, nil
}

func SetUserHomeDir(newHome string) {
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", newHome) //nolint:errcheck
	}
	// *nix
	os.Setenv("HOME", newHome) //nolint:errcheck
}

func RestoreHome(oldHome string) {
	os.Setenv("HOME", oldHome) //nolint:errcheck
}

func ConfigureEnv(region string, defaultRegion string, profile string) {
	os.Setenv("AWS_REGION", region)                //nolint:errcheck
	os.Setenv("AWS_DEFAULT_REGION", defaultRegion) //nolint:errcheck
	os.Setenv("AWS_PROFILE", profile)              //nolint:errcheck
}
