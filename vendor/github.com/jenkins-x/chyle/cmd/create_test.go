package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sirupsen/logrus"
)

func TestCreate(t *testing.T) {
	var code int
	var w sync.WaitGroup

	exitError = func() {
		panic(1)
	}

	exitSuccess = func() {
		panic(0)
	}

	restoreEnvs()
	setenv("CHYLE_GIT_REPOSITORY_PATH", gitRepositoryPath)
	setenv("CHYLE_GIT_REFERENCE_FROM", getCommitFromRef("HEAD~3"))
	setenv("CHYLE_GIT_REFERENCE_TO", getCommitFromRef("test~2^2"))

	w.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				code = r.(int)
			}

			w.Done()
		}()

		os.Args = []string{"", "create"}

		Execute()
	}()

	w.Wait()

	assert.EqualValues(t, 0, code, "Must exit with no errors (exit 0)")
}

func TestCreateWithErrors(t *testing.T) {
	for _, filename := range []string{"../features/init.sh", "../features/merge-commits.sh"} {
		err := exec.Command(filename).Run()

		if err != nil {
			logrus.Fatal(err)
		}
	}

	var code int
	var w sync.WaitGroup

	exitError = func() {
		panic(1)
	}

	exitSuccess = func() {
		panic(0)
	}

	writer = &bytes.Buffer{}

	fixtures := map[string]func(){
		`environment variable missing : "CHYLE_GIT_REPOSITORY_PATH"`: func() {
		},
		`environments variables missing : "CHYLE_GIT_REFERENCE_FROM", "CHYLE_GIT_REFERENCE_TO"`: func() {
			setenv("CHYLE_GIT_REPOSITORY_PATH", "whatever")
		},
		`environment variable missing : "CHYLE_GIT_REFERENCE_TO"`: func() {
			setenv("CHYLE_GIT_REPOSITORY_PATH", "whatever")
			setenv("CHYLE_GIT_REFERENCE_FROM", "ref1")
		},
		`check "whatever" is an existing git repository path`: func() {
			setenv("CHYLE_GIT_REPOSITORY_PATH", "whatever")
			setenv("CHYLE_GIT_REFERENCE_FROM", "ref1")
			setenv("CHYLE_GIT_REFERENCE_TO", "ref2")
		},
		`reference "ref1" can't be found in git repository`: func() {
			setenv("CHYLE_GIT_REPOSITORY_PATH", gitRepositoryPath)
			setenv("CHYLE_GIT_REFERENCE_FROM", "ref1")
			setenv("CHYLE_GIT_REFERENCE_TO", "ref2")
		},
		`reference "ref2" can't be found in git repository`: func() {
			setenv("CHYLE_GIT_REPOSITORY_PATH", gitRepositoryPath)
			setenv("CHYLE_GIT_REFERENCE_FROM", "HEAD")
			setenv("CHYLE_GIT_REFERENCE_TO", "ref2")
		},
	}

	for errStr, fun := range fixtures {
		w.Add(1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					code = r.(int)
				}

				w.Done()
			}()

			restoreEnvs()
			fun()

			os.Args = []string{"", "create"}

			Execute()
		}()

		w.Wait()

		output, err := ioutil.ReadAll(writer.(*bytes.Buffer))

		if err != nil {
			t.Fatal(err)
		}

		assert.EqualValues(t, 1, code, "Must exit with an error (exit 1)")
		assert.Contains(t, string(output), errStr)
	}
}
