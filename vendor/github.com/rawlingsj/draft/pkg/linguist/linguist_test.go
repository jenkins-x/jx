package linguist

import (
	"testing"
)

const (
	appPythonPath   = "testdata/app-python"
	appEmptydirPath = "testdata/app-emptydir"
)

func TestProcessDir(t *testing.T) {
	output, err := ProcessDir(appPythonPath)
	if err != nil {
		t.Error("expected detect to pass")
	}
	if output[0].Language != "Python" {
		t.Errorf("expected output == 'Python', got '%s'", output[0].Language)
	}

	// test with a bad dir
	if _, err := ProcessDir("/dir/does/not/exist"); err == nil {
		t.Error("expected err when running detect with a dir that does not exist")
	}

	// test an application that should fail detection
	output, _ = ProcessDir(appEmptydirPath)
	if len(output) != 0 {
		t.Errorf("expected no languages detected, got '%d'", len(output))
	}
}

func TestGitAttributes(t *testing.T) {
	testCases := []struct {
		path         string
		expectedLang string
	}{
		{"testdata/app-duck", "Duck"},
		{"testdata/app-vendored", "Python"},
		{"testdata/app-not-vendored", "HTML"},
		{"testdata/app-documentation", "Python"},
		{"testdata/app-generated", "Python"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			output, err := ProcessDir(tc.path)
			if err != nil {
				t.Errorf("expected ProcessDir() to pass, got %s", err)
			}
			if output[0].Language != tc.expectedLang {
				t.Errorf("expected output == '%s', got '%s'", tc.expectedLang, output[0].Language)
			}
		})
	}
}

func TestGetAlias(t *testing.T) {
	testcases := map[string]string{
		"maven pom": "Java",
		"mAvEn POM": "Java",
		"c#":        "csharp",
		"Python":    "Python",
	}

	for packName, expectedAlias := range testcases {
		alias := Alias(&Language{Language: packName})
		if alias.Language != expectedAlias {
			t.Errorf("Expected alias to be '%s', got '%s'", expectedAlias, alias.Language)
		}
	}
}
