package cmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// asks to chose from existing projects or optionally creates one if none exist
func (o *CommonOptions) getGoogleProjectId() (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	out, err := o.getCommandOutput("", "gcloud", "projects", "list")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	var existingProjects []string
	for _, l := range lines {
		if strings.Contains(l, clusterListHeader) {
			continue
		}
		fields := strings.Fields(l)
		existingProjects = append(existingProjects, fields[0])
	}

	var projectId string
	if len(existingProjects) == 0 {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("No existing Google Projects exist, create one now?"),
			Default: true,
		}
		flag := true
		err = survey.AskOne(confirm, &flag, nil, surveyOpts)
		if err != nil {
			return "", err
		}
		if !flag {
			return "", errors.New("no google project to create cluster in, please manual create one and rerun this wizard")
		}

		if flag {
			return "", errors.New("auto creating projects not yet implemented, please manually create one and rerun the wizard")
		}
	} else if len(existingProjects) == 1 {
		projectId = existingProjects[0]
		log.Infof("Using the only Google Cloud Project %s to create the cluster\n", util.ColorInfo(projectId))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		if currentProject, err := gke.GetCurrentProject(); err == nil && currentProject != "" {
			prompts.Default = currentProject
		}

		err := survey.AskOne(prompts, &projectId, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}

	if projectId == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}

func (o *CommonOptions) getGoogleZone(projectId string) (string, error) {
	return o.getGoogleZoneWithDefault(projectId, "")
}

func (o *CommonOptions) getGoogleZoneWithDefault(projectId string, defaultZone string) (string, error) {
	availableZones, err := gke.GetGoogleZones(projectId)
	if err != nil {
		return "", err
	}
	prompts := &survey.Select{
		Message:  "Google Cloud Zone:",
		Options:  availableZones,
		PageSize: 10,
		Help:     "The compute zone (e.g. us-central1-a) for the cluster",
		Default:  defaultZone,
	}
	zone := ""
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	err = survey.AskOne(prompts, &zone, nil, surveyOpts)
	if err != nil {
		return "", err
	}
	return zone, nil
}
