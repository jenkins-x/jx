package manifest

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	m := New()
	m.Environments[DefaultEnvironmentName].Name = "foobar"
	expected := "&{foobar    default [] true false 2 [] false []  }"

	actual := fmt.Sprintf("%v", m.Environments[DefaultEnvironmentName])
	if expected != actual {
		t.Errorf("wanted %s, got %s", expected, actual)
	}
}

func TestGenerateName(t *testing.T) {
	name := generateName()
	if name == "" {
		t.Error("expected name to be generated")
	}
	if name != "manifest" {
		t.Errorf("expected name to take the form of the current directory, got %s", name)
	}
}
