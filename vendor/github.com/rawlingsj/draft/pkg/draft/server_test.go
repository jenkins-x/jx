package draft

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestTail tests the "tail" function the draft server uses to retrieve a build's logs.
func TestTail(t *testing.T) {
	// populate content string that represents file content.
	var (
		content  string
		expected string
	)
	const (
		lines = 20 // total lines in "test" file
		limit = 10 // limit number of lines to tail
	)
	for i := 0; i < lines; i++ {
		content += fmt.Sprintf("line%d\n", i)
		if i >= limit {
			expected += fmt.Sprintf("line%d\n", i)
		}
	}
	// create temporary directory
	dir, err := ioutil.TempDir("", "draft-test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)
	// write content to file
	tmp := filepath.Join(dir, "test")
	if err := ioutil.WriteFile(tmp, []byte(content), 0666); err != nil {
		t.Fatalf("failed to write content to file: %v", err)
	}

	// perform the test
	buf := new(bytes.Buffer)
	if err := tail(buf, limit, tmp); err != nil {
		t.Fatalf("failed to tail file %s: %v", tmp, err)
	}
	if result := buf.String(); result != expected {
		t.Fatalf("\nWANT\n%s\nGOT\n%s\n", expected, result)
	}
}
