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