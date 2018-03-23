package jira

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/trivago/tgo/tcontainer"
)

func TestIssueService_Get_Success(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002")

		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	issue, _, err := testClient.Issue.Get("10002", nil)
	if issue == nil {
		t.Error("Expected issue. Issue is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_Get_WithQuerySuccess(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002?expand=foo")

		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	opt := &GetQueryOptions{
		Expand: "foo",
	}
	issue, _, err := testClient.Issue.Get("10002", opt)
	if issue == nil {
		t.Error("Expected issue. Issue is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_Create(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	i := &Issue{
		Fields: &IssueFields{
			Description: "example bug report",
		},
	}
	issue, _, err := testClient.Issue.Create(i)
	if issue == nil {
		t.Error("Expected issue. Issue is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_Update(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/PROJ-9001", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testRequestURL(t, r, "/rest/api/2/issue/PROJ-9001")

		w.WriteHeader(http.StatusNoContent)
	})

	i := &Issue{
		Key: "PROJ-9001",
		Fields: &IssueFields{
			Description: "example bug report",
		},
	}
	issue, _, err := testClient.Issue.Update(i)
	if issue == nil {
		t.Error("Expected issue. Issue is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_UpdateIssue(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/PROJ-9001", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		testRequestURL(t, r, "/rest/api/2/issue/PROJ-9001")

		w.WriteHeader(http.StatusNoContent)
	})
	jID := "PROJ-9001"
	i := make(map[string]interface{})
	fields := make(map[string]interface{})
	i["fields"] = fields
	resp, err := testClient.Issue.UpdateIssue(jID, i)
	if resp == nil {
		t.Error("Expected resp. resp is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}

}

func TestIssueService_AddComment(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/comment", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/comment")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}`)
	})

	c := &Comment{
		Body: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.",
		Visibility: CommentVisibility{
			Type:  "role",
			Value: "Administrators",
		},
	}
	comment, _, err := testClient.Issue.AddComment("10000", c)
	if comment == nil {
		t.Error("Expected Comment. Comment is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_UpdateComment(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/comment/10001", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/comment/10001")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10001","id":"10001","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}`)
	})

	c := &Comment{
		ID:   "10001",
		Body: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.",
		Visibility: CommentVisibility{
			Type:  "role",
			Value: "Administrators",
		},
	}
	comment, _, err := testClient.Issue.UpdateComment("10000", c)
	if comment == nil {
		t.Error("Expected Comment. Comment is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_AddWorklogRecord(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/worklog", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/worklog")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2018-02-14T22:14:46.003+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2018-02-14T22:14:46.003+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}`)
	})
	r := &WorklogRecord{
		TimeSpent: "1h",
	}
	record, _, err := testClient.Issue.AddWorklogRecord("10000", r)
	if record == nil {
		t.Error("Expected Record. Record is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_AddLink(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issueLink", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issueLink")

		w.WriteHeader(http.StatusOK)
	})

	il := &IssueLink{
		Type: IssueLinkType{
			Name: "Duplicate",
		},
		InwardIssue: &Issue{
			Key: "HSP-1",
		},
		OutwardIssue: &Issue{
			Key: "MKY-1",
		},
		Comment: &Comment{
			Body: "Linked related issue!",
			Visibility: CommentVisibility{
				Type:  "group",
				Value: "jira-software-users",
			},
		},
	}
	resp, err := testClient.Issue.AddLink(il)
	if resp == nil {
		t.Error("Expected response. Response is nil")
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected Status code 200. Given %d", resp.StatusCode)
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_Get_Fields(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002")

		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"labels":["test"],"watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"epic": {"id": 19415,"key": "EPIC-77","self": "https://example.atlassian.net/rest/agile/1.0/epic/19415","name": "Epic Name","summary": "Do it","color": {"key": "color_11"},"done": false},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	issue, _, err := testClient.Issue.Get("10002", nil)
	if issue == nil {
		t.Error("Expected issue. Issue is nil")
	}
	if !reflect.DeepEqual(issue.Fields.Labels, []string{"test"}) {
		t.Error("Expected labels for the returned issue")
	}

	if len(issue.Fields.Comments.Comments) != 1 {
		t.Errorf("Expected one comment, %v found", len(issue.Fields.Comments.Comments))
	}
	if issue.Fields.Epic == nil {
		t.Error("Epic expected but not found")
	}

	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_DownloadAttachment(t *testing.T) {
	var testAttachment = "Here is an attachment"

	setup()
	defer teardown()
	testMux.HandleFunc("/secure/attachment/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/secure/attachment/10000/")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testAttachment))
	})

	resp, err := testClient.Issue.DownloadAttachment("10000")
	if resp == nil {
		t.Error("Expected response. Response is nil")
	}
	defer resp.Body.Close()

	attachment, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("Expected attachment text", err)
	}
	if string(attachment) != testAttachment {
		t.Errorf("Expecting an attachment: %s", string(attachment))
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected Status code 200. Given %d", resp.StatusCode)
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_DownloadAttachment_BadStatus(t *testing.T) {

	setup()
	defer teardown()
	testMux.HandleFunc("/secure/attachment/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/secure/attachment/10000/")

		w.WriteHeader(http.StatusForbidden)
	})

	resp, err := testClient.Issue.DownloadAttachment("10000")
	if resp == nil {
		t.Error("Expected response. Response is nil")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected Status code %d. Given %d", http.StatusForbidden, resp.StatusCode)
	}
	if err == nil {
		t.Errorf("Error expected")
	}
}

func TestIssueService_PostAttachment(t *testing.T) {
	var testAttachment = "Here is an attachment"

	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/attachments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/attachments")
		status := http.StatusOK

		file, _, err := r.FormFile("file")
		if err != nil {
			status = http.StatusNotAcceptable
		}
		if file == nil {
			status = http.StatusNoContent
		} else {

			// Read the file into memory
			data, err := ioutil.ReadAll(file)
			if err != nil {
				status = http.StatusInternalServerError
			}
			if string(data) != testAttachment {
				status = http.StatusNotAcceptable
			}

			w.WriteHeader(status)
			fmt.Fprint(w, `[{"self":"http://jira/jira/rest/api/2/attachment/228924","id":"228924","filename":"example.jpg","author":{"self":"http://jira/jira/rest/api/2/user?username=test","name":"test","emailAddress":"test@test.com","avatarUrls":{"16x16":"http://jira/jira/secure/useravatar?size=small&avatarId=10082","48x48":"http://jira/jira/secure/useravatar?avatarId=10082"},"displayName":"Tester","active":true},"created":"2016-05-24T00:25:17.000-0700","size":32280,"mimeType":"image/jpeg","content":"http://jira/jira/secure/attachment/228924/example.jpg","thumbnail":"http://jira/jira/secure/thumbnail/228924/_thumb_228924.png"}]`)
			file.Close()
		}
	})

	reader := strings.NewReader(testAttachment)

	issue, resp, err := testClient.Issue.PostAttachment("10000", reader, "attachment")

	if issue == nil {
		t.Error("Expected response. Response is nil")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected Status code 200. Given %d", resp.StatusCode)
	}

	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_PostAttachment_NoResponse(t *testing.T) {
	var testAttachment = "Here is an attachment"

	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/attachments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/attachments")
		w.WriteHeader(http.StatusOK)
	})
	reader := strings.NewReader(testAttachment)

	_, _, err := testClient.Issue.PostAttachment("10000", reader, "attachment")

	if err == nil {
		t.Errorf("Error expected: %s", err)
	}
}

func TestIssueService_PostAttachment_NoFilename(t *testing.T) {
	var testAttachment = "Here is an attachment"

	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/attachments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/attachments")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"self":"http://jira/jira/rest/api/2/attachment/228924","id":"228924","filename":"example.jpg","author":{"self":"http://jira/jira/rest/api/2/user?username=test","name":"test","emailAddress":"test@test.com","avatarUrls":{"16x16":"http://jira/jira/secure/useravatar?size=small&avatarId=10082","48x48":"http://jira/jira/secure/useravatar?avatarId=10082"},"displayName":"Tester","active":true},"created":"2016-05-24T00:25:17.000-0700","size":32280,"mimeType":"image/jpeg","content":"http://jira/jira/secure/attachment/228924/example.jpg","thumbnail":"http://jira/jira/secure/thumbnail/228924/_thumb_228924.png"}]`)
	})
	reader := strings.NewReader(testAttachment)

	_, _, err := testClient.Issue.PostAttachment("10000", reader, "")

	if err != nil {
		t.Errorf("Error expected: %s", err)
	}
}

func TestIssueService_PostAttachment_NoAttachment(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10000/attachments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/api/2/issue/10000/attachments")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"self":"http://jira/jira/rest/api/2/attachment/228924","id":"228924","filename":"example.jpg","author":{"self":"http://jira/jira/rest/api/2/user?username=test","name":"test","emailAddress":"test@test.com","avatarUrls":{"16x16":"http://jira/jira/secure/useravatar?size=small&avatarId=10082","48x48":"http://jira/jira/secure/useravatar?avatarId=10082"},"displayName":"Tester","active":true},"created":"2016-05-24T00:25:17.000-0700","size":32280,"mimeType":"image/jpeg","content":"http://jira/jira/secure/attachment/228924/example.jpg","thumbnail":"http://jira/jira/secure/thumbnail/228924/_thumb_228924.png"}]`)
	})

	_, _, err := testClient.Issue.PostAttachment("10000", nil, "attachment")

	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_Search(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/search?jql=something&startAt=1&maxResults=40&expand=foo")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"expand": "schema,names","startAt": 1,"maxResults": 40,"total": 6,"issues": [{"expand": "html","id": "10230","self": "http://kelpie9:8081/rest/api/2/issue/BULK-62","key": "BULK-62","fields": {"summary": "testing","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/5","id": "5","description": "The sub-task of the issue","iconUrl": "http://kelpie9:8081/images/icons/issue_subtask.gif","name": "Sub-task","subtask": true},"customfield_10071": null}},{"expand": "html","id": "10004","self": "http://kelpie9:8081/rest/api/2/issue/BULK-47","key": "BULK-47","fields": {"summary": "Cheese v1 2.0 issue","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/3","id": "3","description": "A task that needs to be done.","iconUrl": "http://kelpie9:8081/images/icons/task.gif","name": "Task","subtask": false}}}]}`)
	})

	opt := &SearchOptions{StartAt: 1, MaxResults: 40, Expand: "foo"}
	_, resp, err := testClient.Issue.Search("something", opt)

	if resp == nil {
		t.Errorf("Response given: %+v", resp)
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}

	if resp.StartAt != 1 {
		t.Errorf("StartAt should populate with 1, %v given", resp.StartAt)
	}
	if resp.MaxResults != 40 {
		t.Errorf("StartAt should populate with 40, %v given", resp.MaxResults)
	}
	if resp.Total != 6 {
		t.Errorf("StartAt should populate with 6, %v given", resp.Total)
	}
}

func TestIssueService_Search_WithoutPaging(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/search?jql=something")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"expand": "schema,names","startAt": 0,"maxResults": 50,"total": 6,"issues": [{"expand": "html","id": "10230","self": "http://kelpie9:8081/rest/api/2/issue/BULK-62","key": "BULK-62","fields": {"summary": "testing","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/5","id": "5","description": "The sub-task of the issue","iconUrl": "http://kelpie9:8081/images/icons/issue_subtask.gif","name": "Sub-task","subtask": true},"customfield_10071": null}},{"expand": "html","id": "10004","self": "http://kelpie9:8081/rest/api/2/issue/BULK-47","key": "BULK-47","fields": {"summary": "Cheese v1 2.0 issue","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/3","id": "3","description": "A task that needs to be done.","iconUrl": "http://kelpie9:8081/images/icons/task.gif","name": "Task","subtask": false}}}]}`)
	})

	_, resp, err := testClient.Issue.Search("something", nil)

	if resp == nil {
		t.Errorf("Response given: %+v", resp)
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}

	if resp.StartAt != 0 {
		t.Errorf("StartAt should populate with 0, %v given", resp.StartAt)
	}
	if resp.MaxResults != 50 {
		t.Errorf("StartAt should populate with 50, %v given", resp.MaxResults)
	}
	if resp.Total != 6 {
		t.Errorf("StartAt should populate with 6, %v given", resp.Total)
	}
}

func TestIssueService_SearchPages(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		if r.URL.String() == "/rest/api/2/search?jql=something&startAt=1&maxResults=2&expand=foo&fields=" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"expand": "schema,names","startAt": 1,"maxResults": 2,"total": 6,"issues": [{"expand": "html","id": "10230","self": "http://kelpie9:8081/rest/api/2/issue/BULK-62","key": "BULK-62","fields": {"summary": "testing","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/5","id": "5","description": "The sub-task of the issue","iconUrl": "http://kelpie9:8081/images/icons/issue_subtask.gif","name": "Sub-task","subtask": true},"customfield_10071": null}},{"expand": "html","id": "10004","self": "http://kelpie9:8081/rest/api/2/issue/BULK-47","key": "BULK-47","fields": {"summary": "Cheese v1 2.0 issue","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/3","id": "3","description": "A task that needs to be done.","iconUrl": "http://kelpie9:8081/images/icons/task.gif","name": "Task","subtask": false}}}]}`)
			return
		} else if r.URL.String() == "/rest/api/2/search?jql=something&startAt=3&maxResults=2&expand=foo&fields=" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"expand": "schema,names","startAt": 3,"maxResults": 2,"total": 6,"issues": [{"expand": "html","id": "10230","self": "http://kelpie9:8081/rest/api/2/issue/BULK-62","key": "BULK-62","fields": {"summary": "testing","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/5","id": "5","description": "The sub-task of the issue","iconUrl": "http://kelpie9:8081/images/icons/issue_subtask.gif","name": "Sub-task","subtask": true},"customfield_10071": null}},{"expand": "html","id": "10004","self": "http://kelpie9:8081/rest/api/2/issue/BULK-47","key": "BULK-47","fields": {"summary": "Cheese v1 2.0 issue","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/3","id": "3","description": "A task that needs to be done.","iconUrl": "http://kelpie9:8081/images/icons/task.gif","name": "Task","subtask": false}}}]}`)
			return
		} else if r.URL.String() == "/rest/api/2/search?jql=something&startAt=5&maxResults=2&expand=foo&fields=" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"expand": "schema,names","startAt": 5,"maxResults": 2,"total": 6,"issues": [{"expand": "html","id": "10230","self": "http://kelpie9:8081/rest/api/2/issue/BULK-62","key": "BULK-62","fields": {"summary": "testing","timetracking": null,"issuetype": {"self": "http://kelpie9:8081/rest/api/2/issuetype/5","id": "5","description": "The sub-task of the issue","iconUrl": "http://kelpie9:8081/images/icons/issue_subtask.gif","name": "Sub-task","subtask": true},"customfield_10071": null}}]}`)
			return
		}

		t.Errorf("Unexpected URL: %v", r.URL)
	})

	opt := &SearchOptions{StartAt: 1, MaxResults: 2, Expand: "foo"}
	issues := make([]Issue, 0)
	err := testClient.Issue.SearchPages("something", opt, func(issue Issue) error {
		issues = append(issues, issue)
		return nil
	})

	if err != nil {
		t.Errorf("Error given: %s", err)
	}

	if len(issues) != 5 {
		t.Errorf("Expected 5 issues, %v given", len(issues))
	}
}

func TestIssueService_GetCustomFields(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002")
		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"customfield_123":"test","watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	issue, _, err := testClient.Issue.GetCustomFields("10002")
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
	if issue == nil {
		t.Error("Expected Customfields")
	}
	cf := issue["customfield_123"]
	if cf != "test" {
		t.Error("Expected \"test\" for custom field")
	}
}

func TestIssueService_GetComplexCustomFields(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002")
		fmt.Fprint(w, `{"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations","id":"10002","self":"http://www.example.com/jira/rest/api/2/issue/10002","key":"EX-1","fields":{"customfield_123":{"self":"http://www.example.com/jira/rest/api/2/customFieldOption/123","value":"test","id":"123"},"watcher":{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]},"attachment":[{"self":"http://www.example.com/jira/rest/api/2.0/attachments/10000","filename":"picture.jpg","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred","24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred","32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.461+0000","size":23123,"mimeType":"image/jpeg","content":"http://www.example.com/jira/attachments/10000","thumbnail":"http://www.example.com/jira/secure/thumbnail/10000"}],"sub-tasks":[{"id":"10000","type":{"id":"10000","name":"","inward":"Parent","outward":"Sub-task"},"outwardIssue":{"id":"10003","key":"EX-2","self":"http://www.example.com/jira/rest/api/2/issue/EX-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"description":"example bug report","project":{"self":"http://www.example.com/jira/rest/api/2/project/EX","id":"10000","key":"EX","name":"Example","avatarUrls":{"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000","24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000","16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000","32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"},"projectCategory":{"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000","id":"10000","name":"FIRST","description":"First Project Category"}},"comment":{"comments":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000","id":"10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.","updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","visibility":{"type":"role","value":"Administrators"}}]},"issuelinks":[{"id":"10001","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"outwardIssue":{"id":"10004L","key":"PRJ-2","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}},{"id":"10002","type":{"id":"10000","name":"Dependent","inward":"depends on","outward":"is depended by"},"inwardIssue":{"id":"10004","key":"PRJ-3","self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3","fields":{"status":{"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png","name":"Open"}}}}],"worklog":{"worklogs":[{"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"comment":"I did some work here.","updated":"2016-03-16T04:22:37.471+0000","visibility":{"type":"group","value":"jira-developers"},"started":"2016-03-16T04:22:37.471+0000","timeSpent":"3h 20m","timeSpentSeconds":12000,"id":"100028","issueId":"10002"}]},"updated":"2016-04-06T02:36:53.594-0700","duedate":"2018-01-19","timetracking":{"originalEstimate":"10m","remainingEstimate":"3m","timeSpent":"6m","originalEstimateSeconds":600,"remainingEstimateSeconds":200,"timeSpentSeconds":400}},"names":{"watcher":"watcher","attachment":"attachment","sub-tasks":"sub-tasks","description":"description","project":"project","comment":"comment","issuelinks":"issuelinks","worklog":"worklog","updated":"updated","timetracking":"timetracking"},"schema":{}}`)
	})

	issue, _, err := testClient.Issue.GetCustomFields("10002")
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
	if issue == nil {
		t.Error("Expected Customfields")
	}
	cf := issue["customfield_123"]
	if cf != "test" {
		t.Error("Expected \"test\" for custom field")
	}
}

func TestIssueService_GetTransitions(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/api/2/issue/123/transitions"

	raw, err := ioutil.ReadFile("./mocks/transitions.json")
	if err != nil {
		t.Error(err.Error())
	}

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEndpoint)
		fmt.Fprint(w, string(raw))
	})

	transitions, _, err := testClient.Issue.GetTransitions("123")

	if err != nil {
		t.Errorf("Got error: %v", err)
	}

	if transitions == nil {
		t.Error("Expected transition list. Got nil.")
	}

	if len(transitions) != 2 {
		t.Errorf("Expected 2 transitions. Got %d", len(transitions))
	}

	if transitions[0].Fields["summary"].Required != false {
		t.Errorf("First transition summary field should not be required")
	}
}

func TestIssueService_DoTransition(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/api/2/issue/123/transitions"

	transitionID := "22"

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, testAPIEndpoint)

		decoder := json.NewDecoder(r.Body)
		var payload CreateTransitionPayload
		err := decoder.Decode(&payload)
		if err != nil {
			t.Errorf("Got error: %v", err)
		}

		if payload.Transition.ID != transitionID {
			t.Errorf("Expected %s to be in payload, got %s instead", transitionID, payload.Transition.ID)
		}
	})
	_, err := testClient.Issue.DoTransition("123", transitionID)

	if err != nil {
		t.Errorf("Got error: %v", err)
	}
}

func TestIssueService_DoTransitionWithPayload(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/api/2/issue/123/transitions"

	transitionID := "22"

	customPayload := map[string]interface{}{
		"update": map[string]interface{}{
			"comment": []map[string]interface{}{
				{
					"add": map[string]string{
						"body": "Hello World",
					},
				},
			},
		},
		"transition": TransitionPayload{
			ID: transitionID,
		},
	}

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, testAPIEndpoint)

		decoder := json.NewDecoder(r.Body)
		payload := map[string]interface{}{}
		err := decoder.Decode(&payload)
		if err != nil {
			t.Errorf("Got error: %v", err)
		}

		contains := func(key string) bool {
			_, ok := payload[key]
			return ok
		}

		if !contains("update") || !contains("transition") {
			t.Fatalf("Excpected update, transition to be in payload, got %s instead", payload)
		}

		transition, ok := payload["transition"].(map[string]interface{})
		if !ok {
			t.Fatalf("Excpected transition to be in payload, got %s instead", payload["transition"])
		}

		if transition["id"].(string) != transitionID {
			t.Errorf("Expected %s to be in payload, got %s instead", transitionID, transition["id"])
		}
	})
	_, err := testClient.Issue.DoTransitionWithPayload("123", customPayload)

	if err != nil {
		t.Errorf("Got error: %v", err)
	}
}

func TestIssueFields_TestMarshalJSON_PopulateUnknownsSuccess(t *testing.T) {
	data := `{
			"customfield_123":"test",
			"description":"example bug report",
			"project":{
				"self":"http://www.example.com/jira/rest/api/2/project/EX",
				"id":"10000",
				"key":"EX",
				"name":"Example",
				"avatarUrls":{
					"48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000",
					"24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000",
					"16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000",
					"32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"
				},
				"projectCategory":{
					"self":"http://www.example.com/jira/rest/api/2/projectCategory/10000",
					"id":"10000",
					"name":"FIRST",
					"description":"First Project Category"
				}
			},
			"issuelinks":[
				{
					"id":"10001",
					"type":{
					"id":"10000",
					"name":"Dependent",
					"inward":"depends on",
					"outward":"is depended by"
					},
					"outwardIssue":{
					"id":"10004L",
					"key":"PRJ-2",
					"self":"http://www.example.com/jira/rest/api/2/issue/PRJ-2",
					"fields":{
						"status":{
							"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png",
							"name":"Open"
						}
					}
					}
				},
				{
					"id":"10002",
					"type":{
					"id":"10000",
					"name":"Dependent",
					"inward":"depends on",
					"outward":"is depended by"
					},
					"inwardIssue":{
					"id":"10004",
					"key":"PRJ-3",
					"self":"http://www.example.com/jira/rest/api/2/issue/PRJ-3",
					"fields":{
						"status":{
							"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png",
							"name":"Open"
						}
					}
					}
				}
			]

   }`

	i := new(IssueFields)
	err := json.Unmarshal([]byte(data), i)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	if len(i.Unknowns) != 1 {
		t.Errorf("Expected 1 unknown field to be present, received %d", len(i.Unknowns))
	}
	if i.Description != "example bug report" {
		t.Errorf("Expected description to be \"%s\", received \"%s\"", "example bug report", i.Description)
	}

}

func TestIssueFields_MarshalJSON_OmitsEmptyFields(t *testing.T) {
	i := &IssueFields{
		Description: "blahblah",
		Type: IssueType{
			Name: "Story",
		},
		Labels: []string{"aws-docker"},
	}

	rawdata, err := json.Marshal(i)
	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	// convert json to map and see if unset keys are there
	issuef := tcontainer.NewMarshalMap()
	err = json.Unmarshal(rawdata, &issuef)
	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	_, err = issuef.Int("issuetype/avatarId")
	if err == nil {
		t.Error("Expected non nil error, received nil")
	}

	// verify that the field that should be there, is.
	name, err := issuef.String("issuetype/name")
	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	if name != "Story" {
		t.Errorf("Expected Story, received %s", name)
	}

}

func TestIssueFields_MarshalJSON_Success(t *testing.T) {
	i := &IssueFields{
		Description: "example bug report",
		Unknowns: tcontainer.MarshalMap{
			"customfield_123": "test",
		},
		Project: Project{
			Self: "http://www.example.com/jira/rest/api/2/project/EX",
			ID:   "10000",
			Key:  "EX",
		},
	}

	bytes, err := json.Marshal(i)
	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	received := new(IssueFields)
	// the order of json might be different. so unmarshal it again and compare objects
	err = json.Unmarshal(bytes, received)
	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	if !reflect.DeepEqual(i, received) {
		t.Errorf("Received object different from expected. Expected %+v, received %+v", i, received)
	}
}

func TestInitIssueWithMetaAndFields_Success(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["summary"] = map[string]interface{}{
		"name": "Summary",
		"schema": map[string]interface{}{
			"type": "string",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}
	expectedSummary := "Issue Summary"
	fieldConfig := map[string]string{
		"Summary": "Issue Summary",
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	gotSummary, found := issue.Fields.Unknowns["summary"]
	if !found {
		t.Errorf("Expected summary to be set in issue. Not set.")
	}

	if gotSummary != expectedSummary {
		t.Errorf("Expected %s received %s", expectedSummary, gotSummary)
	}
}

func TestInitIssueWithMetaAndFields_ArrayValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["component"] = map[string]interface{}{
		"name": "Component/s",
		"schema": map[string]interface{}{
			"type":  "array",
			"items": "component",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedComponent := "Jira automation"
	fieldConfig := map[string]string{
		"Component/s": expectedComponent,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	c, isArray := issue.Fields.Unknowns["component"].([]Component)
	if isArray == false {
		t.Error("Expected array, non array object received")
	}

	if len(c) != 1 {
		t.Errorf("Expected received array to be of length 1. Got %d", len(c))
	}

	gotComponent := c[0].Name

	if err != nil {
		t.Errorf("Expected err to be nil, received %s", err)
	}

	if gotComponent != expectedComponent {
		t.Errorf("Expected %s received %s", expectedComponent, gotComponent)
	}
}

func TestInitIssueWithMetaAndFields_DateValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["created"] = map[string]interface{}{
		"name": "Created",
		"schema": map[string]interface{}{
			"type": "date",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedCreated := "19 oct 2012"
	fieldConfig := map[string]string{
		"Created": expectedCreated,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	gotCreated, err := issue.Fields.Unknowns.String("created")
	if err != nil {
		t.Errorf("Expected err to be nil, received %s", err)
	}

	if gotCreated != expectedCreated {
		t.Errorf("Expected %s received %s", expectedCreated, gotCreated)
	}
}

func TestInitIssueWithMetaAndFields_UserValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["assignee"] = map[string]interface{}{
		"name": "Assignee",
		"schema": map[string]interface{}{
			"type": "user",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedAssignee := "jdoe"
	fieldConfig := map[string]string{
		"Assignee": expectedAssignee,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	a, _ := issue.Fields.Unknowns.Value("assignee")
	gotAssignee := a.(User).Name

	if gotAssignee != expectedAssignee {
		t.Errorf("Expected %s received %s", expectedAssignee, gotAssignee)
	}
}

func TestInitIssueWithMetaAndFields_ProjectValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["project"] = map[string]interface{}{
		"name": "Project",
		"schema": map[string]interface{}{
			"type": "project",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	setProject := "somewhere"
	fieldConfig := map[string]string{
		"Project": setProject,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	a, _ := issue.Fields.Unknowns.Value("project")
	gotProject := a.(Project).Name

	if gotProject != metaProject.Name {
		t.Errorf("Expected %s received %s", metaProject.Name, gotProject)
	}
}

func TestInitIssueWithMetaAndFields_PriorityValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["priority"] = map[string]interface{}{
		"name": "Priority",
		"schema": map[string]interface{}{
			"type": "priority",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedPriority := "Normal"
	fieldConfig := map[string]string{
		"Priority": expectedPriority,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	a, _ := issue.Fields.Unknowns.Value("priority")
	gotPriority := a.(Priority).Name

	if gotPriority != expectedPriority {
		t.Errorf("Expected %s received %s", expectedPriority, gotPriority)
	}
}

func TestInitIssueWithMetaAndFields_SelectList(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["someitem"] = map[string]interface{}{
		"name": "A Select Item",
		"schema": map[string]interface{}{
			"type": "option",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedVal := "Value"
	fieldConfig := map[string]string{
		"A Select Item": expectedVal,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	a, _ := issue.Fields.Unknowns.Value("someitem")
	gotVal := a.(Option).Value

	if gotVal != expectedVal {
		t.Errorf("Expected %s received %s", expectedVal, gotVal)
	}
}

func TestInitIssueWithMetaAndFields_IssuetypeValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["issuetype"] = map[string]interface{}{
		"name": "Issue type",
		"schema": map[string]interface{}{
			"type": "issuetype",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	expectedIssuetype := "Bug"
	fieldConfig := map[string]string{
		"Issue type": expectedIssuetype,
	}

	issue, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	a, _ := issue.Fields.Unknowns.Value("issuetype")
	gotIssuetype := a.(IssueType).Name

	if gotIssuetype != expectedIssuetype {
		t.Errorf("Expected %s received %s", expectedIssuetype, gotIssuetype)
	}
}

func TestInitIssueWithmetaAndFields_FailureWithUnknownValueType(t *testing.T) {
	metaProject := MetaProject{
		Name: "Engineering - Dept",
		Id:   "ENG",
	}

	fields := tcontainer.NewMarshalMap()
	fields["issuetype"] = map[string]interface{}{
		"name": "Issue type",
		"schema": map[string]interface{}{
			"type": "randomType",
		},
	}

	metaIssueType := MetaIssueType{
		Fields: fields,
	}

	fieldConfig := map[string]string{
		"Issue tyoe": "sometype",
	}
	_, err := InitIssueWithMetaAndFields(&metaProject, &metaIssueType, fieldConfig)
	if err == nil {
		t.Error("Expected non nil error, received nil")
	}

}

func TestIssueService_Delete(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testRequestURL(t, r, "/rest/api/2/issue/10002")

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprint(w, `{}`)
	})

	resp, err := testClient.Issue.Delete("10002")
	if resp.StatusCode != 204 {
		t.Error("Expected issue not deleted.")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_GetWorklogs(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002/worklog", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002/worklog")

		fmt.Fprint(w, `{"startAt": 1,"maxResults": 40,"total": 1,"worklogs": [{"id": "3","self": "http://kelpie9:8081/rest/api/2/issue/10002/worklog/3","author":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"updateAuthor":{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false},"created":"2016-03-16T04:22:37.356+0000","updated":"2016-03-16T04:22:37.356+0000","comment":"","started":"2016-03-16T04:22:37.356+0000","timeSpent": "1h","timeSpentSeconds": 3600,"issueId":"10002"}]}`)
	})

	worklog, _, err := testClient.Issue.GetWorklogs("10002")
	if worklog == nil {
		t.Error("Expected worklog. Worklog is nil")
	}

	if len(worklog.Worklogs) != 1 {
		t.Error("Expected 1 worklog")
	}

	if worklog.Worklogs[0].Author.Name != "fred" {
		t.Error("Expected worklog author to be fred")
	}

	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestIssueService_GetWatchers(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/api/2/issue/10002/watchers", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/issue/10002/watchers")

		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers","isWatching":false,"watchCount":1,"watchers":[{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","name":"fred","displayName":"Fred F. User","active":false}]}`)
	})

	testMux.HandleFunc("/rest/api/2/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, "/rest/api/2/user?username=fred")

		fmt.Fprint(w, `{"self":"http://www.example.com/jira/rest/api/2/user?username=fred","key":"fred",
        "name":"fred","emailAddress":"fred@example.com","avatarUrls":{"48x48":"http://www.example.com/jira/secure/useravatar?size=large&ownerId=fred",
        "24x24":"http://www.example.com/jira/secure/useravatar?size=small&ownerId=fred","16x16":"http://www.example.com/jira/secure/useravatar?size=xsmall&ownerId=fred",
        "32x32":"http://www.example.com/jira/secure/useravatar?size=medium&ownerId=fred"},"displayName":"Fred F. User","active":true,"timeZone":"Australia/Sydney","groups":{"size":3,"items":[
        {"name":"jira-user","self":"http://www.example.com/jira/rest/api/2/group?groupname=jira-user"},{"name":"jira-admin",
        "self":"http://www.example.com/jira/rest/api/2/group?groupname=jira-admin"},{"name":"important","self":"http://www.example.com/jira/rest/api/2/group?groupname=important"
        }]},"applicationRoles":{"size":1,"items":[]},"expand":"groups,applicationRoles"}`)
	})

	watchers, _, err := testClient.Issue.GetWatchers("10002")
	if err != nil {
		t.Errorf("Error given: %s", err)
		return
	}
	if watchers == nil {
		t.Error("Expected watchers. Watchers is nil")
		return
	}
	if len(*watchers) != 1 {
		t.Errorf("Expected 1 watcher, got: %d", len(*watchers))
		return
	}
	if (*watchers)[0].Name != "fred" {
		t.Error("Expected watcher name fred")
	}
}
