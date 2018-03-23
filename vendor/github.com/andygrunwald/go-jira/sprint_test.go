package jira

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestSprintService_MoveIssuesToSprint(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/agile/1.0/sprint/123/issue"

	issuesToMove := []string{"KEY-1", "KEY-2"}

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, testAPIEndpoint)

		decoder := json.NewDecoder(r.Body)
		var payload IssuesWrapper
		err := decoder.Decode(&payload)
		if err != nil {
			t.Errorf("Got error: %v", err)
		}

		if payload.Issues[0] != issuesToMove[0] {
			t.Errorf("Expected %s to be in payload, got %s instead", issuesToMove[0], payload.Issues[0])
		}
	})
	_, err := testClient.Sprint.MoveIssuesToSprint(123, issuesToMove)

	if err != nil {
		t.Errorf("Got error: %v", err)
	}
}

func TestSprintService_GetIssuesForSprint(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/agile/1.0/sprint/123/issue"

	raw, err := ioutil.ReadFile("./mocks/issues_in_sprint.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, string(raw))
	})

	issues, _, err := testClient.Sprint.GetIssuesForSprint(123)
	if err != nil {
		t.Errorf("Error given: %v", err)
	}
	if issues == nil {
		t.Error("Expected issues in sprint list. Issues list is nil")
	}
	if len(issues) != 1 {
		t.Errorf("Expect there to be 1 issue in the sprint, found %v", len(issues))
	}

}
