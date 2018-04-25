package gitlab

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestListTags(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/1/repository/tags", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"name": "1.0.0"},{"name": "1.0.1"}]`)
	})

	opt := &ListTagsOptions{Page: 2, PerPage: 3}

	tags, _, err := client.Tags.ListTags(1, opt)
	if err != nil {
		t.Errorf("Tags.ListTags returned error: %v", err)
	}

	want := []*Tag{{Name: "1.0.0"}, {Name: "1.0.1"}}
	if !reflect.DeepEqual(want, tags) {
		t.Errorf("Tags.ListTags returned %+v, want %+v", tags, want)
	}
}

func TestCreateRelease(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/1/repository/tags/1.0.0/release", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, `{"tag_name": "1.0.0", "description": "Amazing release. Wow"}`)
	})

	opt := &CreateReleaseOptions{Description: String("Amazing release. Wow")}

	release, _, err := client.Tags.CreateRelease(1, "1.0.0", opt)
	if err != nil {
		t.Errorf("Tags.CreateRelease returned error: %v", err)
	}

	want := &Release{TagName: "1.0.0", Description: "Amazing release. Wow"}
	if !reflect.DeepEqual(want, release) {
		t.Errorf("Tags.CreateRelease returned %+v, want %+v", release, want)
	}
}

func TestUpdateRelease(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/1/repository/tags/1.0.0/release", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		fmt.Fprint(w, `{"tag_name": "1.0.0", "description": "Amazing release. Wow!"}`)
	})

	opt := &UpdateReleaseOptions{Description: String("Amazing release. Wow!")}

	release, _, err := client.Tags.UpdateRelease(1, "1.0.0", opt)
	if err != nil {
		t.Errorf("Tags.UpdateRelease returned error: %v", err)
	}

	want := &Release{TagName: "1.0.0", Description: "Amazing release. Wow!"}
	if !reflect.DeepEqual(want, release) {
		t.Errorf("Tags.UpdateRelease returned %+v, want %+v", release, want)
	}
}
