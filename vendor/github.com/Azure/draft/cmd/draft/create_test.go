package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/testing/helpers"
)

const gitkeepfile = ".gitkeep"

func TestCreate(t *testing.T) {
	var generatedpath = "testdata/create/generated"

	testCases := []struct {
		src         string
		expectedErr error
	}{
		{"testdata/create/src/empty", nil},
		{"testdata/create/src/html-but-actually-go", nil},
		{"testdata/create/src/simple-go", nil},
		{"testdata/create/src/simple-go-with-draftignore", nil},
		{"testdata/create/src/simple-go-with-chart", nil},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("create %s", tc.src), func(t *testing.T) {
			pDir, teardown := tempDir(t, "draft-create")
			defer teardown()

			destcompare := filepath.Join(generatedpath, path.Base(tc.src))
			helpers.CopyTree(t, tc.src, pDir)
			// Test
			create := &createCmd{
				appName: "myapp",
				out:     ioutil.Discard,
				home:    draftpath.Home("testdata/drafthome/"),
				dest:    pDir,
			}
			err := create.run()

			// Error checking
			if err != tc.expectedErr {
				t.Errorf("draft create returned an unexpected error: '%v'", err)
				return
			}

			// append .gitkeep file on empty directories when we expect `draft create` to pass
			if tc.expectedErr == nil {
				addGitKeep(t, pDir)
			}

			// Compare directories to ensure they are identical
			assertIdentical(t, pDir, destcompare)
		})
	}
}

func TestNormalizeApplicationName(t *testing.T) {
	testCases := []string{
		"AppName",
		"appName",
		"appname",
	}

	expected := "appname"

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("normalizeApplicationName %s", tc), func(t *testing.T) {
			create := &createCmd{
				appName: tc,
				out:     os.Stdout,
				home:    draftpath.Home("../../"),
				dest:    "",
			}

			create.normalizeApplicationName()
			assertEqualString(t, create.appName, expected)
		})
	}
}

// tempDir create and clean a temporary directory to work in our tests
func tempDir(t *testing.T, description string) (string, func()) {
	path, err := ioutil.TempDir("", description)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return path, func() {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

// add .gitkeep to generated empty directories
func addGitKeep(t *testing.T, p string) {
	if err := filepath.Walk(p, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		files, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			f, err := os.OpenFile(filepath.Join(p, gitkeepfile), os.O_RDONLY|os.O_CREATE, 0666)
			if err != nil {
				return err
			}
			defer f.Close()
		}
		return nil
	}); err != nil {
		t.Fatalf("couldn't stamp git keep files: %v", err)
	}
}

// Compares two strings and asserts equivalence.
func assertEqualString(t *testing.T, is string, shouldBe string) {
	if is == shouldBe {
		return
	}

	t.Fatalf("Assertion failed: Expected: %s. Got: %s", shouldBe, is)
}

// assertIdentical compares recursively all original and generated file content
func assertIdentical(t *testing.T, original, generated string) {
	if err := filepath.Walk(original, func(f string, fi os.FileInfo, err error) error {
		relp := strings.TrimPrefix(f, original)
		// root path
		if relp == "" {
			return nil
		}
		relp = relp[1:]
		p := filepath.Join(generated, relp)

		// .keep files are only for keeping directory creations in remote git repo
		if filepath.Base(p) == gitkeepfile {
			return nil
		}

		fo, err := os.Stat(p)
		if err != nil {
			t.Fatalf("%s doesn't exist while %s does", p, f)
		}

		if fi.IsDir() {
			if !fo.IsDir() {
				t.Fatalf("%s is a directory and %s isn't", f, p)
			}
			// else, it's a directory as well and we are done.
			return nil
		}

		wanted, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Couldn't read %s: %v", f, err)
		}
		actual, err := ioutil.ReadFile(p)
		if err != nil {
			t.Fatalf("Couldn't read %s: %v", p, err)
		}
		if !bytes.Equal(actual, wanted) {
			t.Errorf("%s and %s content differs:\nACTUAL:\n%s\n\nWANTED:\n%s", p, f, actual, wanted)
		}
		return nil
	}); err != nil {
		t.Fatalf("err: %s", err)
	}

	// on the other side, check that all generated items are in origin
	if err := filepath.Walk(generated, func(f string, _ os.FileInfo, err error) error {
		relp := strings.TrimPrefix(f, generated)
		// root path
		if relp == "" {
			return nil
		}
		relp = relp[1:]
		p := filepath.Join(original, relp)

		// .keep files are only for keeping directory creations in remote git repo
		if filepath.Base(p) == gitkeepfile {
			return nil
		}

		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s doesn't exist while %s does", p, f)
		}
		return nil
	}); err != nil {
		t.Fatalf("err: %s", err)
	}
}
