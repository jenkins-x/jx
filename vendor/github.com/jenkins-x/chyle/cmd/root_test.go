package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	var code int
	var w sync.WaitGroup

	exitError = func() {
		panic(1)
	}

	exitSuccess = func() {
		panic(0)
	}

	writer = &bytes.Buffer{}

	w.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				code = r.(int)
			}

			w.Done()
		}()

		os.Args = []string{"", "whatever"}

		Execute()
	}()

	w.Wait()

	output, err := ioutil.ReadAll(writer.(*bytes.Buffer))

	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(output), `unknown command "whatever" for "chyle"`)
	assert.EqualValues(t, 1, code, "Must exit with an errors (exit 1)")
}
