//
// Copyright 2017, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestDisableRunner(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/1/runners/2", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Runners.DisableProjectRunner(1, 2, nil)
	if err != nil {
		t.Fatalf("Runners.DisableProjectRunner returns an error: %v", err)
	}
}

func TestListRunnersJobs(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/runners/1/jobs", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1},{"id":2}]`)
	})

	opt := &ListRunnerJobsOptions{}

	jobs, _, err := client.Runners.ListRunnerJobs(1, opt)
	if err != nil {
		t.Fatalf("Runners.ListRunnersJobs returns an error: %v", err)
	}

	want := []*Job{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(want, jobs) {
		t.Errorf("Runners.ListRunnersJobs returned %+v, want %+v", jobs, want)
	}
}

func TestRemoveRunner(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/runners/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Runners.RemoveRunner(1, nil)
	if err != nil {
		t.Fatalf("Runners.RemoveARunner returns an error: %v", err)
	}
}

const exampleDetailRsp = `{
	"active": true,
	"architecture": null,
	"description": "test-1-20150125-test",
	"id": 6,
	"is_shared": false,
	"contacted_at": "2016-01-25T16:39:48.066Z",
	"name": null,
	"online": true,
	"status": "online",
	"platform": null,
	"projects": [
		{
			"id": 1,
			"name": "GitLab Community Edition",
			"name_with_namespace": "GitLab.org / GitLab Community Edition",
			"path": "gitlab-ce",
			"path_with_namespace": "gitlab-org/gitlab-ce"
		}
	],
	"token": "205086a8e3b9a2b818ffac9b89d102",
	"revision": null,
	"tag_list": [
		"ruby",
		"mysql"
	],
	"version": null,
	"access_level": "ref_protected"
}`

func TestUpdateRunnersDetails(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/runners/6", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		fmt.Fprint(w, exampleDetailRsp)
	})

	opt := &UpdateRunnerDetailsOptions{}

	details, _, err := client.Runners.UpdateRunnerDetails(6, opt, nil)
	if err != nil {
		t.Fatalf("Runners.UpdateRunnersDetails returns an error: %v", err)
	}

	want := expectedParsedDetails()
	if !reflect.DeepEqual(want, details) {
		t.Errorf("Runners.UpdateRunnersDetails returned %+v, want %+v", details, want)
	}
}

func TestGetRunnerDetails(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/runners/6", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, exampleDetailRsp)
	})

	details, _, err := client.Runners.GetRunnerDetails(6, nil)
	if err != nil {
		t.Fatalf("Runners.GetRunnerDetails returns an error: %v", err)
	}

	want := expectedParsedDetails()
	if !reflect.DeepEqual(want, details) {
		t.Errorf("Runners.UpdateRunnersDetails returned %+v, want %+v", details, want)
	}
}

// helper function returning expected result for string: &exampleDetailRsp
func expectedParsedDetails() *RunnerDetails {
	proj := struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		NameWithNamespace string `json:"name_with_namespace"`
		Path              string `json:"path"`
		PathWithNamespace string `json:"path_with_namespace"`
	}{ID: 1, Name: "GitLab Community Edition", NameWithNamespace: "GitLab.org / GitLab Community Edition", Path: "gitlab-ce", PathWithNamespace: "gitlab-org/gitlab-ce"}
	timestamp, _ := time.Parse("2006-01-02T15:04:05.000Z", "2016-01-25T16:39:48.066Z")
	return &RunnerDetails{Active: true, Description: "test-1-20150125-test", ID: 6, IsShared: false, ContactedAt: &timestamp, Online: true, Status: "online", Token: "205086a8e3b9a2b818ffac9b89d102", TagList: []string{"ruby", "mysql"}, AccessLevel: "ref_protected", Projects: []struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		NameWithNamespace string `json:"name_with_namespace"`
		Path              string `json:"path"`
		PathWithNamespace string `json:"path_with_namespace"`
	}{proj}}
}
