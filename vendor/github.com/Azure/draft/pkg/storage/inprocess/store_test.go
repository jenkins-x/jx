package inprocess

import (
	"context"
	"github.com/Azure/draft/pkg/storage"
	"github.com/golang/protobuf/ptypes"
	"reflect"
	"testing"
)

func TestStoreDeleteBuilds(t *testing.T) {
	var (
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	builds, err := store.DeleteBuilds(ctx, "app1")
	if err != nil {
		t.Fatalf("failed to delete build entries: %v", err)
	}
	if len(store.builds["app1"]) > 0 {
		t.Fatal("expected build entries to empty")
	}
	assertEqual(t, "DeleteBuilds", builds, []*storage.Object{
		objectStub("foo1", "bar1", []byte("foobar1")),
		objectStub("foo2", "bar2", []byte("foobar2")),
		objectStub("foo3", "bar3", []byte("foobar3")),
		objectStub("foo4", "bar4", []byte("foobar4")),
	})
}

func TestStoreDeleteBuild(t *testing.T) {
	var (
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	build, err := store.DeleteBuild(ctx, "app1", "foo1")
	if err != nil {
		t.Fatalf("failed to delete build entry: %v", err)
	}
	assertEqual(t, "DeleteBuild", build, objectStub("foo1", "bar1", []byte("foobar1")))
}

func TestStoreCreateBuild(t *testing.T) {
	var (
		build = objectStub("foo", "bar", []byte("foobar"))
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	if err := store.CreateBuild(ctx, "app2", build); err != nil {
		t.Fatalf("failed to create storage entry: %v", err)
	}
	alt, err := store.GetBuild(ctx, "app2", build.BuildID)
	if err != nil {
		t.Fatalf("failed to get build entry: %v", err)
	}
	assertEqual(t, "CreateBuild", build, alt)

	// try creating a second time; this should fail with ErrAppStorageExists.
	if err := store.CreateBuild(ctx, "app2", build); err == nil {
		t.Fatalf("expected second CreateBuild to fail")
	}
}

func TestStoreUpdateBuild(t *testing.T) {
	var (
		build = objectStub("foo", "bar", []byte("foobar"))
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	if err := store.UpdateBuild(ctx, "app2", build); err != nil {
		t.Fatalf("failed to update storage entry: %v", err)
	}
	alt, err := store.GetBuild(ctx, "app2", build.BuildID)
	if err != nil {
		t.Fatalf("failed to get build entry: %v", err)
	}
	assertEqual(t, "UpdateBuild", build, alt)
}

func TestStoreGetBuilds(t *testing.T) {
	var (
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	// make sure the build is returnable by appID
	ls, err := store.GetBuilds(ctx, "app1")
	if err != nil {
		t.Fatalf("could not get builds: %v", err)
	}
	assertEqual(t, "GetBuilds", ls, []*storage.Object{
		objectStub("foo1", "bar1", []byte("foobar1")),
		objectStub("foo2", "bar2", []byte("foobar2")),
		objectStub("foo3", "bar3", []byte("foobar3")),
		objectStub("foo4", "bar4", []byte("foobar4")),
	})
	// try fetching a build with an unknown appID; should fail.
	if alt, err := store.GetBuilds(ctx, "bad"); err == nil {
		t.Fatalf("want err != nil; got alt: %+v", alt)
	}
}

func TestStoreGetBuild(t *testing.T) {
	var (
		store = NewStoreWithMocks()
		ctx   = context.TODO()
	)
	// make sure the build is returnable by appID
	obj, err := store.GetBuild(ctx, "app1", "foo1")
	if err != nil {
		t.Fatalf("could not get build: %v", err)
	}
	assertEqual(t, "GetBuild", obj, objectStub("foo1", "bar1", []byte("foobar1")))
	// try fetching a build with an unknown appID; should fail.
	if alt, err := store.GetBuild(ctx, "bad", ""); err == nil {
		t.Fatalf("want err != nil; got alt: %+v", alt)
	}
}

func NewStoreWithMocks() *Store {
	store := NewStore()
	store.builds["app1"] = []*storage.Object{
		objectStub("foo1", "bar1", []byte("foobar1")),
		objectStub("foo2", "bar2", []byte("foobar2")),
		objectStub("foo3", "bar3", []byte("foobar3")),
		objectStub("foo4", "bar4", []byte("foobar4")),
	}
	return store
}

func assertEqual(t *testing.T, label string, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("failed equality for %s", label)
	}
}

func objectStub(buildID, release string, contextID []byte) *storage.Object {
	return &storage.Object{
		BuildID:   buildID,
		Release:   release,
		ContextID: contextID,
		CreatedAt: createdAt,
	}
}

var createdAt = ptypes.TimestampNow()
