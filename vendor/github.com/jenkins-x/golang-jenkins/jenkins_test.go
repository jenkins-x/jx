package gojenkins

import (
	"fmt"
	"testing"
	"time"
)

func NewJenkinsWithTestData() *Jenkins {
	auth := Auth{
		Username: "admin",
		ApiToken: "admin",
	}
	//return NewJenkins(&auth, "http://jenkins.default.192.168.64.22.nip.io/")
	return NewJenkins(&auth, "http://example.com")
}

func Test(t *testing.T) {
	jenkins := NewJenkinsWithTestData()
	jobs, err := jenkins.GetJobs()

	if err != nil {
		t.Errorf("error %v\n", err)
	}

	if len(jobs) == 0 {
		t.Errorf("return no jobs\n")
	}
}

func TestAddJobToView(t *testing.T) {
	jenkins := NewJenkinsWithTestData()

	scm := Scm{
		Class: "hudson.scm.SubversionSCM",
	}
	jobItem := JobItem{
		MavenJobItem: &MavenJobItem{
			Plugin:               "maven-plugin@2.7.1",
			Description:          "test description",
			Scm:                  scm,
			Triggers:             Triggers{},
			RunPostStepsIfResult: RunPostStepsIfResult{},
			Settings:             JobSettings{Class: "jenkins.mvn.DefaultSettingsProvider"},
			GlobalSettings:       JobSettings{Class: "jenkins.mvn.DefaultSettingsProvider"},
		},
	}
	newJobName := fmt.Sprintf("test-with-view-%d", time.Now().UnixNano())
	newViewName := fmt.Sprintf("test-view-%d", time.Now().UnixNano())
	jenkins.CreateJob(jobItem, newJobName)
	jenkins.CreateView(NewListView(newViewName))

	job := Job{Name: newJobName}
	err := jenkins.AddJobToView(newViewName, job)

	if err != nil {
		t.Errorf("error %v\n", err)
	}
}

func TestCreateView(t *testing.T) {
	jenkins := NewJenkinsWithTestData()

	newViewName := fmt.Sprintf("test-view-%d", time.Now().UnixNano())
	err := jenkins.CreateView(NewListView(newViewName))

	if err != nil {
		t.Errorf("error %v\n", err)
	}
}

func TestCreateCredential(t *testing.T) {
	jenkins := NewJenkinsWithTestData()
	err := jenkins.CreateCredential("test6", "test5", "tester")
	if err != nil {
		t.Errorf("error %v\n", err)
	}
}

func TestCreateJobItem(t *testing.T) {
	jenkins := NewJenkinsWithTestData()
	scm := Scm{
		ScmContent: ScmSvn{
			Locations: Locations{
				[]ScmSvnLocation{
					ScmSvnLocation{IgnoreExternalsOption: "false", DepthOption: "infinity", Local: ".", Remote: "http://some-svn-url"},
				},
			},
			IgnoreDirPropChanges: "false",
			FilterChangelog:      "false",
			WorkspaceUpdater:     WorkspaceUpdater{Class: "hudson.scm.subversion.UpdateUpdater"},
		},
		Class:  "hudson.scm.SubversionSCM",
		Plugin: "subversion@1.54",
	}
	triggers := Triggers{[]Trigger{ScmTrigger{}}}
	postStep := RunPostStepsIfResult{Name: "FAILURE", Ordinal: "2", Color: "RED", CompleteBuild: "true"}
	settings := JobSettings{Class: "jenkins.mvn.DefaultSettingsProvider"}
	globalSettings := JobSettings{Class: "jenkins.mvn.DefaultSettingsProvider"}
	jobItem := JobItem{
		MavenJobItem: &MavenJobItem{
			Plugin:               "maven-plugin@2.7.1",
			Description:          "test description",
			Scm:                  scm,
			Triggers:             triggers,
			RunPostStepsIfResult: postStep,
			Settings:             settings,
			GlobalSettings:       globalSettings,
		},
	}

	newJobName := fmt.Sprintf("test-%d", time.Now().UnixNano())
	err := jenkins.CreateJob(jobItem, newJobName)

	if err != nil {
		t.Errorf("error %v\n", err)
	}

	jobs, _ := jenkins.GetJobs()
	foundNewJob := false
	for _, v := range jobs {
		if v.Name == newJobName {
			foundNewJob = true
		}
	}

	if !foundNewJob {
		t.Errorf("error %s not found\n", newJobName)
	}
}
