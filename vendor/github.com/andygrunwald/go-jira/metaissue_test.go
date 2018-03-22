package jira

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIssueService_GetCreateMeta_Success(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/api/2/issue/createmeta"

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEndpoint)

		fmt.Fprint(w, `{
	"expand": "projects",
	"projects": [{
		"expand": "issuetypes",
		"self": "https://my.jira.com/rest/api/2/project/11300",
		"id": "11300",
		"key": "SPN",
		"name": "Super Project Name",
		"avatarUrls": {
			"48x48": "https://my.jira.com/secure/projectavatar?pid=11300&avatarId=14405",
			"24x24": "https://my.jira.com/secure/projectavatar?size=small&pid=11300&avatarId=14405",
			"16x16": "https://my.jira.com/secure/projectavatar?size=xsmall&pid=11300&avatarId=14405",
			"32x32": "https://my.jira.com/secure/projectavatar?size=medium&pid=11300&avatarId=14405"
		},
		"issuetypes": [{
			"self": "https://my.jira.com/rest/api/2/issuetype/6",
			"id": "6",
			"description": "An issue which ideally should be able to be completed in one step",
			"iconUrl": "https://my.jira.com/secure/viewavatar?size=xsmall&avatarId=14006&avatarType=issuetype",
			"name": "Request",
			"subtask": false,
			"expand": "fields",
			"fields": {
				"summary": {
					"required": true,
					"schema": {
						"type": "string",
						"system": "summary"
					},
					"name": "Summary",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"issuetype": {
					"required": true,
					"schema": {
						"type": "issuetype",
						"system": "issuetype"
					},
					"name": "Issue Type",
					"hasDefaultValue": false,
					"operations": [

					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/issuetype/6",
						"id": "6",
						"description": "An issue which ideally should be able to be completed in one step",
						"iconUrl": "https://my.jira.com/secure/viewavatar?size=xsmall&avatarId=14006&avatarType=issuetype",
						"name": "Request",
						"subtask": false,
						"avatarId": 14006
					}]
				},
				"components": {
					"required": true,
					"schema": {
						"type": "array",
						"items": "component",
						"system": "components"
					},
					"name": "Component/s",
					"hasDefaultValue": false,
					"operations": [
						"add",
						"set",
						"remove"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/component/14144",
						"id": "14144",
						"name": "Build automation",
						"description": "Jenkins, webhooks, etc."
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14149",
						"id": "14149",
						"name": "Caches and noSQL",
						"description": "Cassandra, Memcached, Redis, Twemproxy, Xcache"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14152",
						"id": "14152",
						"name": "Cloud services",
						"description": "AWS and similar services"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14147",
						"id": "14147",
						"name": "Code quality tools",
						"description": "Code sniffer, Sonar"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14156",
						"id": "14156",
						"name": "Configuration management and provisioning",
						"description": "Apache/PHP modules, Consul, Salt"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/13606",
						"id": "13606",
						"name": "Cronjobs",
						"description": "Cronjobs in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14150",
						"id": "14150",
						"name": "Data pipelines and queues",
						"description": "Kafka, RabbitMq"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14159",
						"id": "14159",
						"name": "Database",
						"description": "MySQL related problems"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14314",
						"id": "14314",
						"name": "Documentation"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14151",
						"id": "14151",
						"name": "Git",
						"description": "Bitbucket, GitHub, GitLab, Git in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14155",
						"id": "14155",
						"name": "HTTP services",
						"description": "CDN, HaProxy, HTTP, Varnish"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14154",
						"id": "14154",
						"name": "Job and service scheduling",
						"description": "Chronos, Docker, Marathon, Mesos"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14158",
						"id": "14158",
						"name": "Legacy",
						"description": "Everything related to legacy"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14157",
						"id": "14157",
						"name": "Monitoring",
						"description": "Collectd, Nagios, Monitoring in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14148",
						"id": "14148",
						"name": "Other services"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/13602",
						"id": "13602",
						"name": "Package management",
						"description": "Composer, Medusa, Satis"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14145",
						"id": "14145",
						"name": "Release",
						"description": "Directory config, release queries, rewrite rules"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14146",
						"id": "14146",
						"name": "Staging systems and VMs",
						"description": "Stage, QA machines, KVMs,Vagrant"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14153",
						"id": "14153",
						"name": "Blog"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14143",
						"id": "14143",
						"name": "Test automation",
						"description": "Testing infrastructure in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14221",
						"id": "14221",
						"name": "Internal Infrastructure"
					}]
				},
				"attachment": {
					"required": false,
					"schema": {
						"type": "array",
						"items": "attachment",
						"system": "attachment"
					},
					"name": "Attachment",
					"hasDefaultValue": false,
					"operations": [

					]
				},
				"duedate": {
					"required": false,
					"schema": {
						"type": "date",
						"system": "duedate"
					},
					"name": "Due Date",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"description": {
					"required": false,
					"schema": {
						"type": "string",
						"system": "description"
					},
					"name": "Description",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"customfield_10806": {
					"required": false,
					"schema": {
						"type": "any",
						"custom": "com.pyxis.greenhopper.jira:gh-epic-link",
						"customId": 10806
					},
					"name": "Epic Link",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"project": {
					"required": true,
					"schema": {
						"type": "project",
						"system": "project"
					},
					"name": "Project",
					"hasDefaultValue": false,
					"operations": [
						"set"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/project/11300",
						"id": "11300",
						"key": "SPN",
						"name": "Super Project Name",
						"avatarUrls": {
							"48x48": "https://my.jira.com/secure/projectavatar?pid=11300&avatarId=14405",
							"24x24": "https://my.jira.com/secure/projectavatar?size=small&pid=11300&avatarId=14405",
							"16x16": "https://my.jira.com/secure/projectavatar?size=xsmall&pid=11300&avatarId=14405",
							"32x32": "https://my.jira.com/secure/projectavatar?size=medium&pid=11300&avatarId=14405"
						},
						"projectCategory": {
							"self": "https://my.jira.com/rest/api/2/projectCategory/10100",
							"id": "10100",
							"description": "",
							"name": "Product & Development"
						}
					}]
				},
				"assignee": {
					"required": true,
					"schema": {
						"type": "user",
						"system": "assignee"
					},
					"name": "Assignee",
					"autoCompleteUrl": "https://my.jira.com/rest/api/latest/user/assignable/search?issueKey=null&username=",
					"hasDefaultValue": true,
					"operations": [
						"set"
					]
				},
				"priority": {
					"required": false,
					"schema": {
						"type": "priority",
						"system": "priority"
					},
					"name": "Priority",
					"hasDefaultValue": true,
					"operations": [
						"set"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/priority/1",
						"iconUrl": "https://my.jira.com/images/icons/priorities/blocker.svg",
						"name": "Immediate",
						"id": "1"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/2",
						"iconUrl": "https://my.jira.com/images/icons/priorities/critical.svg",
						"name": "Urgent",
						"id": "2"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/3",
						"iconUrl": "https://my.jira.com/images/icons/priorities/major.svg",
						"name": "High",
						"id": "3"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/6",
						"iconUrl": "https://my.jira.com/images/icons/priorities/moderate.svg",
						"name": "Moderate",
						"id": "6"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/4",
						"iconUrl": "https://my.jira.com/images/icons/priorities/minor.svg",
						"name": "Normal",
						"id": "4"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/5",
						"iconUrl": "https://my.jira.com/images/icons/priorities/trivial.svg",
						"name": "Low",
						"id": "5"
					}]
				},
				"labels": {
					"required": false,
					"schema": {
						"type": "array",
						"items": "string",
						"system": "labels"
					},
					"name": "Labels",
					"autoCompleteUrl": "https://my.jira.com/rest/api/1.0/labels/suggest?query=",
					"hasDefaultValue": false,
					"operations": [
						"add",
						"set",
						"remove"
					]
				}
			}
		}]
	}]
    }`)
	})

	issue, _, err := testClient.Issue.GetCreateMeta("SPN")
	if err != nil {
		t.Errorf("Expected nil error but got %s", err)
	}

	if len(issue.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(issue.Projects))
	}
	for _, project := range issue.Projects {
		if len(project.IssueTypes) != 1 {
			t.Errorf("Expected 1 issueTypes, got %d", len(project.IssueTypes))
		}
		for _, issueTypes := range project.IssueTypes {
			requiredFields := 0
			fields := issueTypes.Fields
			for _, value := range fields {
				for key, value := range value.(map[string]interface{}) {
					if key == "required" && value == true {
						requiredFields = requiredFields + 1
					}
				}

			}
			if requiredFields != 5 {
				t.Errorf("Expected 5 required fields from Create Meta information, got %d", requiredFields)
			}
		}
	}

}

func TestMetaIssueType_GetCreateMetaWithOptions(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/api/2/issue/createmeta"

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEndpoint)

		fmt.Fprint(w, `{
	"expand": "projects",
	"projects": [{
		"expand": "issuetypes",
		"self": "https://my.jira.com/rest/api/2/project/11300",
		"id": "11300",
		"key": "SPN",
		"name": "Super Project Name",
		"avatarUrls": {
			"48x48": "https://my.jira.com/secure/projectavatar?pid=11300&avatarId=14405",
			"24x24": "https://my.jira.com/secure/projectavatar?size=small&pid=11300&avatarId=14405",
			"16x16": "https://my.jira.com/secure/projectavatar?size=xsmall&pid=11300&avatarId=14405",
			"32x32": "https://my.jira.com/secure/projectavatar?size=medium&pid=11300&avatarId=14405"
		},
		"issuetypes": [{
			"self": "https://my.jira.com/rest/api/2/issuetype/6",
			"id": "6",
			"description": "An issue which ideally should be able to be completed in one step",
			"iconUrl": "https://my.jira.com/secure/viewavatar?size=xsmall&avatarId=14006&avatarType=issuetype",
			"name": "Request",
			"subtask": false,
			"expand": "fields",
			"fields": {
				"summary": {
					"required": true,
					"schema": {
						"type": "string",
						"system": "summary"
					},
					"name": "Summary",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"issuetype": {
					"required": true,
					"schema": {
						"type": "issuetype",
						"system": "issuetype"
					},
					"name": "Issue Type",
					"hasDefaultValue": false,
					"operations": [

					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/issuetype/6",
						"id": "6",
						"description": "An issue which ideally should be able to be completed in one step",
						"iconUrl": "https://my.jira.com/secure/viewavatar?size=xsmall&avatarId=14006&avatarType=issuetype",
						"name": "Request",
						"subtask": false,
						"avatarId": 14006
					}]
				},
				"components": {
					"required": true,
					"schema": {
						"type": "array",
						"items": "component",
						"system": "components"
					},
					"name": "Component/s",
					"hasDefaultValue": false,
					"operations": [
						"add",
						"set",
						"remove"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/component/14144",
						"id": "14144",
						"name": "Build automation",
						"description": "Jenkins, webhooks, etc."
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14149",
						"id": "14149",
						"name": "Caches and noSQL",
						"description": "Cassandra, Memcached, Redis, Twemproxy, Xcache"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14152",
						"id": "14152",
						"name": "Cloud services",
						"description": "AWS and similar services"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14147",
						"id": "14147",
						"name": "Code quality tools",
						"description": "Code sniffer, Sonar"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14156",
						"id": "14156",
						"name": "Configuration management and provisioning",
						"description": "Apache/PHP modules, Consul, Salt"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/13606",
						"id": "13606",
						"name": "Cronjobs",
						"description": "Cronjobs in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14150",
						"id": "14150",
						"name": "Data pipelines and queues",
						"description": "Kafka, RabbitMq"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14159",
						"id": "14159",
						"name": "Database",
						"description": "MySQL related problems"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14314",
						"id": "14314",
						"name": "Documentation"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14151",
						"id": "14151",
						"name": "Git",
						"description": "Bitbucket, GitHub, GitLab, Git in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14155",
						"id": "14155",
						"name": "HTTP services",
						"description": "CDN, HaProxy, HTTP, Varnish"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14154",
						"id": "14154",
						"name": "Job and service scheduling",
						"description": "Chronos, Docker, Marathon, Mesos"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14158",
						"id": "14158",
						"name": "Legacy",
						"description": "Everything related to legacy"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14157",
						"id": "14157",
						"name": "Monitoring",
						"description": "Collectd, Nagios, Monitoring in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14148",
						"id": "14148",
						"name": "Other services"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/13602",
						"id": "13602",
						"name": "Package management",
						"description": "Composer, Medusa, Satis"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14145",
						"id": "14145",
						"name": "Release",
						"description": "Directory config, release queries, rewrite rules"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14146",
						"id": "14146",
						"name": "Staging systems and VMs",
						"description": "Stage, QA machines, KVMs,Vagrant"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14153",
						"id": "14153",
						"name": "Blog"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14143",
						"id": "14143",
						"name": "Test automation",
						"description": "Testing infrastructure in general"
					}, {
						"self": "https://my.jira.com/rest/api/2/component/14221",
						"id": "14221",
						"name": "Internal Infrastructure"
					}]
				},
				"attachment": {
					"required": false,
					"schema": {
						"type": "array",
						"items": "attachment",
						"system": "attachment"
					},
					"name": "Attachment",
					"hasDefaultValue": false,
					"operations": [

					]
				},
				"duedate": {
					"required": false,
					"schema": {
						"type": "date",
						"system": "duedate"
					},
					"name": "Due Date",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"description": {
					"required": false,
					"schema": {
						"type": "string",
						"system": "description"
					},
					"name": "Description",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"customfield_10806": {
					"required": false,
					"schema": {
						"type": "any",
						"custom": "com.pyxis.greenhopper.jira:gh-epic-link",
						"customId": 10806
					},
					"name": "Epic Link",
					"hasDefaultValue": false,
					"operations": [
						"set"
					]
				},
				"project": {
					"required": true,
					"schema": {
						"type": "project",
						"system": "project"
					},
					"name": "Project",
					"hasDefaultValue": false,
					"operations": [
						"set"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/project/11300",
						"id": "11300",
						"key": "SPN",
						"name": "Super Project Name",
						"avatarUrls": {
							"48x48": "https://my.jira.com/secure/projectavatar?pid=11300&avatarId=14405",
							"24x24": "https://my.jira.com/secure/projectavatar?size=small&pid=11300&avatarId=14405",
							"16x16": "https://my.jira.com/secure/projectavatar?size=xsmall&pid=11300&avatarId=14405",
							"32x32": "https://my.jira.com/secure/projectavatar?size=medium&pid=11300&avatarId=14405"
						},
						"projectCategory": {
							"self": "https://my.jira.com/rest/api/2/projectCategory/10100",
							"id": "10100",
							"description": "",
							"name": "Product & Development"
						}
					}]
				},
				"assignee": {
					"required": true,
					"schema": {
						"type": "user",
						"system": "assignee"
					},
					"name": "Assignee",
					"autoCompleteUrl": "https://my.jira.com/rest/api/latest/user/assignable/search?issueKey=null&username=",
					"hasDefaultValue": true,
					"operations": [
						"set"
					]
				},
				"priority": {
					"required": false,
					"schema": {
						"type": "priority",
						"system": "priority"
					},
					"name": "Priority",
					"hasDefaultValue": true,
					"operations": [
						"set"
					],
					"allowedValues": [{
						"self": "https://my.jira.com/rest/api/2/priority/1",
						"iconUrl": "https://my.jira.com/images/icons/priorities/blocker.svg",
						"name": "Immediate",
						"id": "1"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/2",
						"iconUrl": "https://my.jira.com/images/icons/priorities/critical.svg",
						"name": "Urgent",
						"id": "2"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/3",
						"iconUrl": "https://my.jira.com/images/icons/priorities/major.svg",
						"name": "High",
						"id": "3"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/6",
						"iconUrl": "https://my.jira.com/images/icons/priorities/moderate.svg",
						"name": "Moderate",
						"id": "6"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/4",
						"iconUrl": "https://my.jira.com/images/icons/priorities/minor.svg",
						"name": "Normal",
						"id": "4"
					}, {
						"self": "https://my.jira.com/rest/api/2/priority/5",
						"iconUrl": "https://my.jira.com/images/icons/priorities/trivial.svg",
						"name": "Low",
						"id": "5"
					}]
				},
				"labels": {
					"required": false,
					"schema": {
						"type": "array",
						"items": "string",
						"system": "labels"
					},
					"name": "Labels",
					"autoCompleteUrl": "https://my.jira.com/rest/api/1.0/labels/suggest?query=",
					"hasDefaultValue": false,
					"operations": [
						"add",
						"set",
						"remove"
					]
				}
			}
		}]
	}]
    }`)
	})

	issue, _, err := testClient.Issue.GetCreateMetaWithOptions(&GetQueryOptions{Expand: "projects.issuetypes.fields"})
	if err != nil {
		t.Errorf("Expected nil error but got %s", err)
	}

	if len(issue.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(issue.Projects))
	}
	for _, project := range issue.Projects {
		if len(project.IssueTypes) != 1 {
			t.Errorf("Expected 1 issueTypes, got %d", len(project.IssueTypes))
		}
		for _, issueTypes := range project.IssueTypes {
			requiredFields := 0
			fields := issueTypes.Fields
			for _, value := range fields {
				for key, value := range value.(map[string]interface{}) {
					if key == "required" && value == true {
						requiredFields = requiredFields + 1
					}
				}

			}
			if requiredFields != 5 {
				t.Errorf("Expected 5 required fields from Create Meta information, got %d", requiredFields)
			}
		}
	}
}

func TestMetaIssueType_GetMandatoryFields(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
		"name":     "Summary",
	}

	data["components"] = map[string]interface{}{
		"required": true,
		"name":     "Components",
	}

	data["epicLink"] = map[string]interface{}{
		"required": false,
		"name":     "Epic Link",
	}

	m := new(MetaIssueType)
	m.Fields = data

	mandatory, err := m.GetMandatoryFields()
	if err != nil {
		t.Errorf("Expected nil error, received %s", err)
	}

	if len(mandatory) != 2 {
		t.Errorf("Expected 2 received %+v", mandatory)
	}
}

func TestMetaIssueType_GetMandatoryFields_NonExistentRequiredKey_Fail(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"name": "Summary",
	}

	m := new(MetaIssueType)
	m.Fields = data

	_, err := m.GetMandatoryFields()
	if err == nil {
		t.Error("Expected non nil errpr, received nil")
	}
}

func TestMetaIssueType_GetMandatoryFields_NonExistentNameKey_Fail(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
	}

	m := new(MetaIssueType)
	m.Fields = data

	_, err := m.GetMandatoryFields()
	if err == nil {
		t.Error("Expected non nil errpr, received nil")
	}
}

func TestMetaIssueType_GetAllFields(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
		"name":     "Summary",
	}

	data["components"] = map[string]interface{}{
		"required": true,
		"name":     "Components",
	}

	data["epicLink"] = map[string]interface{}{
		"required": false,
		"name":     "Epic Link",
	}

	m := new(MetaIssueType)
	m.Fields = data

	mandatory, err := m.GetAllFields()

	if err != nil {
		t.Errorf("Expected nil err, received %s", err)
	}

	if len(mandatory) != 3 {
		t.Errorf("Expected 3 received %+v", mandatory)
	}
}

func TestMetaIssueType_GetAllFields_NonExistingNameKey_Fail(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
	}

	m := new(MetaIssueType)
	m.Fields = data

	_, err := m.GetAllFields()
	if err == nil {
		t.Error("Expected non nil error, received nil")
	}
}

func TestMetaIssueType_CheckCompleteAndAvailable_MandatoryMissing(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
		"name":     "Summary",
	}

	data["someKey"] = map[string]interface{}{
		"required": false,
		"name":     "SomeKey",
	}

	config := map[string]string{
		"SomeKey": "somevalue",
	}

	m := new(MetaIssueType)
	m.Fields = data

	ok, err := m.CheckCompleteAndAvailable(config)
	if err == nil {
		t.Error("Expected non nil error. Received nil")
	}

	if ok != false {
		t.Error("Expected false, got true")
	}

}

func TestMetaIssueType_CheckCompleteAndAvailable_NotAvailable(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
		"name":     "Summary",
	}

	config := map[string]string{
		"Summary": "Issue Summary",
		"SomeKey": "somevalue",
	}

	m := new(MetaIssueType)
	m.Fields = data

	ok, err := m.CheckCompleteAndAvailable(config)
	if err == nil {
		t.Error("Expected non nil error. Received nil")
	}

	if ok != false {
		t.Error("Expected false, got true")
	}

}

func TestMetaIssueType_CheckCompleteAndAvailable_Success(t *testing.T) {
	data := make(map[string]interface{})

	data["summary"] = map[string]interface{}{
		"required": true,
		"name":     "Summary",
	}

	data["someKey"] = map[string]interface{}{
		"required": false,
		"name":     "SomeKey",
	}

	config := map[string]string{
		"SomeKey": "somevalue",
		"Summary": "Issue summary",
	}

	m := new(MetaIssueType)
	m.Fields = data

	ok, err := m.CheckCompleteAndAvailable(config)
	if err != nil {
		t.Errorf("Expected nil error. Received %s", err)
	}

	if ok != true {
		t.Error("Expected true, got false")
	}

}

func TestCreateMetaInfo_GetProjectWithName_Success(t *testing.T) {
	metainfo := new(CreateMetaInfo)
	metainfo.Projects = append(metainfo.Projects, &MetaProject{
		Name: "SPN",
	})

	project := metainfo.GetProjectWithName("SPN")
	if project == nil {
		t.Errorf("Expected non nil value, received nil")
	}
}

func TestMetaProject_GetIssueTypeWithName_CaseMismatch_Success(t *testing.T) {
	m := new(MetaProject)
	m.IssueTypes = append(m.IssueTypes, &MetaIssueType{
		Name: "Bug",
	})

	issuetype := m.GetIssueTypeWithName("BUG")

	if issuetype == nil {
		t.Errorf("Expected non nil value, received nil")
	}
}

func TestCreateMetaInfo_GetProjectWithKey_Success(t *testing.T) {
	metainfo := new(CreateMetaInfo)
	metainfo.Projects = append(metainfo.Projects, &MetaProject{
		Key: "SPNKEY",
	})

	project := metainfo.GetProjectWithKey("SPNKEY")
	if project == nil {
		t.Errorf("Expected non nil value, received nil")
	}
}

func TestCreateMetaInfo_GetProjectWithKey_NilForNonExistent(t *testing.T) {
	metainfo := new(CreateMetaInfo)
	metainfo.Projects = append(metainfo.Projects, &MetaProject{
		Key: "SPNKEY",
	})

	project := metainfo.GetProjectWithKey("SPN")
	if project != nil {
		t.Errorf("Expected nil, received value")
	}
}
