package cmd

import (
	"github.com/jenkins-x/golang-jenkins"
)

// getAllPipelineJobNames returns all the pipeline job names
func (o *CommonOptions) getAllPipelineJobNames(jenkinsClient gojenkins.JenkinsClient, jobNames *[]string, jobName string) error {
	job, err := jenkinsClient.GetJob(jobName)
	if err != nil {
		return err
	}
	if len(job.Jobs) == 0 {
		*jobNames = append(*jobNames, job.FullName)
	}
	for _, j := range job.Jobs {
		err = o.getAllPipelineJobNames(jenkinsClient, jobNames, job.FullName+"/"+j.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
