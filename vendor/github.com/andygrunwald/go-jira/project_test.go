package jira

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestProjectService_GetList(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/api/2/project"

	raw, err := ioutil.ReadFile("./mocks/all_projects.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, string(raw))
	})

	projects, _, err := testClient.Project.GetList()
	if projects == nil {
		t.Error("Expected project list. Project list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestProjectService_ListWithOptions(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/api/2/project"

	raw, err := ioutil.ReadFile("./mocks/all_projects.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/project?expand=issueTypes")
		fmt.Fprint(w, string(raw))
	})

	projects, _, err := testClient.Project.ListWithOptions(&GetQueryOptions{Expand: "issueTypes"})
	if projects == nil {
		t.Error("Expected project list. Project list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}


func TestProjectService_Get(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/api/2/project/12310505"

	raw, err := ioutil.ReadFile("./mocks/project.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, string(raw))
	})

	projects, _, err := testClient.Project.Get("12310505")
	if projects == nil {
		t.Error("Expected project list. Project list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestProjectService_Get_NoProject(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/api/2/project/99999999"

	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, nil)
	})

	projects, resp, err := testClient.Project.Get("99999999")
	if projects != nil {
		t.Errorf("Expected nil. Got %+v", projects)
	}

	if resp.Status == "404" {
		t.Errorf("Expected status 404. Got %s", resp.Status)
	}
	if err == nil {
		t.Errorf("Error given: %s", err)
	}
}
