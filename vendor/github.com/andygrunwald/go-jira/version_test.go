package jira

import (
	"fmt"
	"net/http"
	"testing"
)

func TestVersionService_Get_Success(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/version/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/version/10002")

		fmt.Fprint(w, `{
			"self": "http://www.example.com/jira/rest/api/2/version/10002",
			"id": "10002",
			"description": "An excellent version",
			"name": "New Version 1",
			"archived": false,
			"released": true,
			"releaseDate": "2010-07-06",
			"overdue": true,
			"userReleaseDate": "6/Jul/2010",
			"projectId": 10000
		}`)
	})

	version, _, err := testClient.Version.Get(10002)
	if version == nil {
		t.Error("Expected version. Issue is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestVersionService_Create(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/version", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/version")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"description": "An excellent version",
			"name": "New Version 1",
			"archived": false,
			"released": true,
			"releaseDate": "2010-07-06",
			"userReleaseDate": "6/Jul/2010",
			"project": "PXA",
			"projectId": 10000
		  }`)
	})

	v := &Version{
		Name:            "New Version 1",
		Description:     "An excellent version",
		ProjectID:       10000,
		Released:        true,
		ReleaseDate:     "2010-07-06",
		UserReleaseDate: "6/Jul/2010",
	}

	version, _, err := testClient.Version.Create(v)
	if version == nil {
		t.Error("Expected version. Version is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestServiceService_Update(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/version/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testRequestURL(t, r, "/rest/api/2/version/10002")
		fmt.Fprint(w, `{
			"description": "An excellent updated version",
			"name": "New Updated Version 1",
			"archived": false,
			"released": true,
			"releaseDate": "2010-07-06",
			"userReleaseDate": "6/Jul/2010",
			"project": "PXA",
			"projectId": 10000
		  }`)
	})

	v := &Version{
		ID:          "10002",
		Name:        "New Updated Version 1",
		Description: "An excellent updated version",
	}

	version, _, err := testClient.Version.Update(v)
	if version == nil {
		t.Error("Expected version. Version is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}
