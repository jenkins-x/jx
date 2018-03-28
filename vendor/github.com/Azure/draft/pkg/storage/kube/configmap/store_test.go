package configmap

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/draft/pkg/storage"
)

func TestStoreDeleteBuilds(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
	)
	switch objs, err := store.DeleteBuilds(ctx, "app1"); {
	case err != nil:
		t.Fatalf("failed to delete builds: %v", err)
	case len(objs) != 4:
		t.Fatalf("expected 4 deleted builds, got %d", len(objs))
	}
}

func TestStoreDeleteBuild(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
	)
	obj, err := store.DeleteBuild(ctx, "app1", "foo4")
	if err != nil {
		t.Fatalf("failed to delete build: %v", err)
	}
	assertEqual(t, "DeleteBuild", obj, objectStub("foo4", "bar4", []byte("foobar4")))
}

func TestStoreCreateBuild(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
	)
	obj := objectStub("foo1", "bar1", []byte("foobar1"))
	err := store.CreateBuild(ctx, "app2", obj)
	if err != nil {
		t.Fatalf("failed to create build: %v", err)
	}
	got, err := store.GetBuild(ctx, "app2", "foo1")
	if err != nil {
		t.Fatalf("failed to get storage object: %v", err)
	}
	assertEqual(t, "CreateBuild", got, obj)
}

func TestStoreUpdateBuild(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
	)
	obj := objectStub("foo1", "bar1", []byte("foobar1"))
	err := store.UpdateBuild(ctx, "app2", obj)
	if err != nil {
		t.Fatalf("failed to update build: %v", err)
	}
	got, err := store.GetBuild(ctx, "app2", "foo1")
	if err != nil {
		t.Fatalf("failed to get storage object: %v", err)
	}
	assertEqual(t, "UpdateBuild", got, obj)
}

func TestStoreGetBuilds(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
	)
	switch got, err := store.GetBuilds(ctx, "app1"); {
	case err != nil:
		t.Fatalf("failed to get builds: %v", err)
	case len(got) != 4:
		t.Fatalf("expected 4 storage objects, got %d", len(got))
	}
}

func TestStoreGetBuild(t *testing.T) {
	var (
		store = newMockConfigMapsTestFixture(t)
		ctx   = context.Background()
		want  = objectStub("foo1", "bar1", []byte("foobar1"))
	)
	got, err := store.GetBuild(ctx, "app1", "foo1")
	if err != nil {
		t.Fatalf("failed to get storage object: %v", err)
	}
	assertEqual(t, "GetBuild", got, want)
}

//
// test fixtures / helpers
//

func newMockConfigMapsTestFixture(t *testing.T) *ConfigMaps {
	var mocks = []struct {
		appName string
		objects []*storage.Object
	}{
		{
			appName: "app1",
			objects: []*storage.Object{
				objectStub("foo1", "bar1", []byte("foobar1")),
				objectStub("foo2", "bar2", []byte("foobar2")),
				objectStub("foo3", "bar3", []byte("foobar3")),
				objectStub("foo4", "bar4", []byte("foobar4")),
			},
		},
	}
	return NewConfigMapsWithMocks(t, mocks...)
}

func objectStub(buildID, release string, contextID []byte) *storage.Object {
	return &storage.Object{
		BuildID:   buildID,
		Release:   release,
		ContextID: contextID,
	}
}

func assertEqual(t *testing.T, label string, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("failed equality for %s", label)
	}
}
