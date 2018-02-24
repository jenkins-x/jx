package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/antham/envh"
)

var gitRepositoryPath = "testing-repository"

var envs map[string]string

func TestMain(m *testing.M) {
	saveExistingEnvs()
	code := m.Run()
	os.Exit(code)
}

func saveExistingEnvs() {
	var err error
	env := envh.NewEnv()

	envs, err = env.FindEntries(".*")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func restoreEnvs() {
	os.Clearenv()

	if len(envs) != 0 {
		for key, value := range envs {
			setenv(key, value)
		}
	}
}

func setenv(key string, value string) {
	err := os.Setenv(key, value)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getCommitFromRef(ref string) string {
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = gitRepositoryPath

	ID, err := cmd.Output()
	ID = ID[:len(ID)-1]

	if err != nil {
		logrus.WithField("ID", string(ID)).Fatal(err)
	}

	return string(ID)
}
