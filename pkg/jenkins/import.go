package jenkins

import (
	"fmt"
	"io"
	"net/url"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"strings"
)

// ImportProject imports a MultiBranchProject into Jeknins for the given git URL
func ImportProject(out io.Writer, jenk *gojenkins.Jenkins, gitURL string, jenkinsfile string, credentials string, failIfExists bool, gitProvider gits.GitProvider, authConfigSvc auth.AuthConfigService) error {
	if gitURL == "" {
		return fmt.Errorf("No Git repository URL found!")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", gitURL, err)
	}

	if credentials == "" {
		credentials = strings.TrimSuffix(DefaultJenkinsCredentialsPrefix+gitInfo.Host, ".com")
	}
	_, err = jenk.GetCredential(credentials)
	if err != nil {
		config := authConfigSvc.Config()
		server := config.GetOrCreateServer(gitInfo.Host)
		user, err := config.PickServerUserAuth(server, "user name for the Jenkins Pipeline")
		if err != nil {
			return err
		}
		err = jenk.CreateCredential(credentials, user.Username, user.ApiToken)
		if err != nil {
			return fmt.Errorf("error creating jenkins credential %s %v", "bdd-test", err)
		}
	}

	org := gitInfo.Organisation
	folder, err := jenk.GetJob(org)
	if err != nil {
		// could not find folder so lets try create it
		jobUrl := util.UrlJoin(jenk.BaseURL(), jenk.GetJobURLPath(org))
		folderXml := CreateFolderXml(jobUrl, org)
		//fmt.Fprintf(out, "XML: %s\n", folderXml)
		err = jenk.CreateJobWithXML(folderXml, org)
		if err != nil {
			return fmt.Errorf("Failed to create the %s folder in jenkins: %s", org, err)
		}
		//fmt.Fprintf(out, "Created Jenkins folder: %s\n", org)
	} else {
		c := folder.Class
		if c != "com.cloudbees.hudson.plugins.folder.Folder" {
			fmt.Fprintf(out, "Warning the folder %s is of class %s", org, c)
		}
	}
	projectXml := CreateMultiBranchProjectXml(gitInfo, credentials, jenkinsfile)
	jobName := gitInfo.Name
	job, err := jenk.GetJobByPath(org, jobName)
	if err == nil {
		if failIfExists {
			return fmt.Errorf("Job already exists in Jenkins at %s", job.Url)
		} else {
			fmt.Fprintf(out, "Job already exists in Jenkins at %s\n", job.Url)
		}
	} else {
		//fmt.Fprintf(out, "Creating MultiBranchProject %s from XML: %s\n", jobName, projectXml)
		err = jenk.CreateFolderJobWithXML(projectXml, org, jobName)
		if err != nil {
			return fmt.Errorf("Failed to create MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
		}
		job, err = jenk.GetJobByPath(org, jobName)
		if err != nil {
			return fmt.Errorf("Failed to find the MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
		}
		fmt.Fprintf(out, "Created Jenkins Project: %s\n", util.ColorInfo(job.Url))
		params := url.Values{}
		err = jenk.Build(job, params)
		if err != nil {
			return fmt.Errorf("Failed to trigger job %s due to %s", job.Url, err)
		}
	}

	// register the webhook
	suffix := gitProvider.JenkinsWebHookPath(gitURL, "")
	webhookUrl := util.UrlJoin(jenk.BaseURL(), suffix)
	webhook := &gits.GitWebHookArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		URL:   webhookUrl,
	}
	return gitProvider.CreateWebHook(webhook)
}
