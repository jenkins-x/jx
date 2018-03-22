package jira

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGroupService_Get(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/group/member", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/group/member?groupname=default")
		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/group/member?includeInactiveUsers=false&maxResults=50&groupname=default&startAt=0","maxResults":50,"startAt":0,"total":2,"isLast":true,"values":[{"self":"http://www.example.com/jira/rest/api/2/user?username=michael","name":"michael","key":"michael","emailAddress":"michael@example.com","displayName":"MichaelScofield","active":true,"timeZone":"Australia/Sydney"},{"self":"http://www.example.com/jira/rest/api/2/user?username=alex","name":"alex","key":"alex","emailAddress":"alex@example.com","displayName":"AlexanderMahone","active":true,"timeZone":"Australia/Sydney"}]}`)
	})
	if members, _, err := testClient.Group.Get("default"); err != nil {
		t.Errorf("Error given: %s", err)
	} else if members == nil {
		t.Error("Expected members. Group.Members is nil")
	}
}

func TestGroupService_Add(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/group/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/group/user?groupname=default")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"name":"default","self":"http://www.example.com/jira/rest/api/2/group?groupname=default","users":{"size":1,"items":[],"max-results":50,"start-index":0,"end-index":0},"expand":"users"}`)
	})

	if group, _, err := testClient.Group.Add("default", "theodore"); err != nil {
		t.Errorf("Error given: %s", err)
	} else if group == nil {
		t.Error("Expected group. Group is nil")
	}
}

func TestGroupService_Remove(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/group/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testRequestURL(t, r, "/rest/api/2/group/user?groupname=default")

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"name":"default","self":"http://www.example.com/jira/rest/api/2/group?groupname=default","users":{"size":1,"items":[],"max-results":50,"start-index":0,"end-index":0},"expand":"users"}`)
	})

	if _, err := testClient.Group.Remove("default", "theodore"); err != nil {
		t.Errorf("Error given: %s", err)
	}
}
