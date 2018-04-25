package pack

import (
	"reflect"
	"testing"
)

func TestListAll(t *testing.T) {
	const packsRoot = "repo/testdata/packs"

	var want = []string{
		"github.com/testOrg1/testRepo1/testpack1",
		"github.com/testOrg1/testRepo1/testpack2",
		"github.com/testOrg1/testRepo2/testpack1",
		"github.com/testOrg1/testRepo2/testpack2",
	}
	got, err := List(packsRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want: %v\ngot: %v\n", want, got)
	}
}

func TestListRepo(t *testing.T) {
	const (
		packsRoot = "repo/testdata/packs"
		packsRepo = "github.com/testOrg1/testRepo1"
	)
	var want = []string{
		"github.com/testOrg1/testRepo1/testpack1",
		"github.com/testOrg1/testRepo1/testpack2",
	}

	got, err := List(packsRoot, packsRepo)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want: %v\ngot: %v\n", want, got)
	}
}
