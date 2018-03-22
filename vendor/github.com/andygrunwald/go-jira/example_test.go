package jira_test

import (
	"crypto/tls"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
)

func ExampleNewClient() {
	jiraClient, _ := jira.NewClient(nil, "https://issues.apache.org/jira/")
	issue, _, _ := jiraClient.Issue.Get("MESOS-3325", nil)

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("Type: %s\n", issue.Fields.Type.Name)
	fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)

	// Output:
	// MESOS-3325: Running mesos-slave@0.23 in a container causes slave to be lost after a restart
	// Type: Bug
	// Priority: Critical
}

func ExampleNewClient_ignoreCertificateErrors() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	jiraClient, _ := jira.NewClient(client, "https://issues.apache.org/jira/")
	issue, _, _ := jiraClient.Issue.Get("MESOS-3325", nil)

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("Type: %s\n", issue.Fields.Type.Name)
	fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)

	// Output:
	// MESOS-3325: Running mesos-slave@0.23 in a container causes slave to be lost after a restart
	// Type: Bug
	// Priority: Critical
}

func ExampleAuthenticationService_SetBasicAuth() {
	jiraClient, err := jira.NewClient(nil, "https://your.jira-instance.com/")
	if err != nil {
		panic(err)
	}
	jiraClient.Authentication.SetBasicAuth("username", "password")

	issue, _, err := jiraClient.Issue.Get("SYS-5156", nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
}

func ExampleAuthenticationService_AcquireSessionCookie() {
	jiraClient, err := jira.NewClient(nil, "https://your.jira-instance.com/")
	if err != nil {
		panic(err)
	}

	res, err := jiraClient.Authentication.AcquireSessionCookie("username", "password")
	if err != nil || res == false {
		fmt.Printf("Result: %v\n", res)
		panic(err)
	}

	issue, _, err := jiraClient.Issue.Get("SYS-5156", nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
}

func ExampleIssueService_Create() {
	jiraClient, err := jira.NewClient(nil, "https://your.jira-instance.com/")
	if err != nil {
		panic(err)
	}

	res, err := jiraClient.Authentication.AcquireSessionCookie("username", "password")
	if err != nil || res == false {
		fmt.Printf("Result: %v\n", res)
		panic(err)
	}

	i := jira.Issue{
		Fields: &jira.IssueFields{
			Assignee: &jira.User{
				Name: "myuser",
			},
			Reporter: &jira.User{
				Name: "youruser",
			},
			Description: "Test Issue",
			Type: jira.IssueType{
				Name: "Bug",
			},
			Project: jira.Project{
				Key: "PROJ1",
			},
			Summary: "Just a demo issue",
		},
	}
	issue, _, err := jiraClient.Issue.Create(&i)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
}

func ExampleClient_Do() {
	jiraClient, _ := jira.NewClient(nil, "https://jira.atlassian.com/")
	req, _ := jiraClient.NewRequest("GET", "/rest/api/2/project", nil)

	projects := new([]jira.Project)
	_, err := jiraClient.Do(req, projects)
	if err != nil {
		panic(err)
	}

	for _, project := range *projects {
		fmt.Printf("%s: %s\n", project.Key, project.Name)
	}
}
