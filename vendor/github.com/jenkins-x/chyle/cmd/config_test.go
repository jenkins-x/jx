package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var code int
	var wg sync.WaitGroup

	exitError = func() {
		panic(1)
	}

	exitSuccess = func() {
		panic(0)
	}

	wg.Add(1)

	reader = bytes.NewBufferString("test\ntest\ntest\nq\n")
	writer = &bytes.Buffer{}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				code = r.(int)
			}

			wg.Done()
		}()

		os.Args = []string{"", "config"}

		Execute()
	}()

	wg.Wait()

	promptRecord, err := ioutil.ReadAll(writer.(*bytes.Buffer))

	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 0, code, "Must exit with no errors (exit 0)")
	assert.Contains(t, string(promptRecord), "Enter a git commit ID that start your range")
	assert.Contains(t, string(promptRecord), `CHYLE_GIT_REFERENCE_TO="test"`)
}
