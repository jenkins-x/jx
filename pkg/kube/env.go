package kube

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"strings"
)

// CreateEnvironmentSurvey creates a Survey on the given environment using the default options
// from the CLI
func CreateEnvironmentSurvey(data *Environment, config *Environment, noGitOps bool, ns string) error {
	name := data.Name
	createMode := name == ""
	if createMode {
		if config.Name != "" {
			data.Name = config.Name
		} else {
			q := &survey.Input{
				Message: "Name:",
				Help: "The Environment name must be unique, lower case and a valid DNS name",
			}
			// TODO validate/transform to match valid kubnernetes names syntax
			err := survey.AskOne(q, &data.Name, survey.Required)
			if err != nil {
				return err
			}
		}
	}
	if config.Spec.Label != "" {
		data.Spec.Label = config.Spec.Label
	} else {
		defaultValue := data.Spec.Label
		if defaultValue == "" {
			defaultValue = strings.Title(data.Name)
		}
		q := &survey.Input{
			Message: "Label:",
			Default: defaultValue,
			Help: "The Environment label is a person friendly descriptive text like 'Staging' or 'Production'",
		}
		err := survey.AskOne(q, &data.Spec.Label, survey.Required)
		if err != nil {
			return err
		}
	}
	if config.Spec.Namespace != "" {
		data.Spec.Namespace = config.Spec.Namespace
	} else {
		defaultValue := data.Spec.Namespace
		if defaultValue == "" {
			// lets use the namespace as a team name
			defaultValue = data.Namespace
			if defaultValue == "" {
				defaultValue = ns
			}
			if data.Name != "" {
				if defaultValue == "" {
					defaultValue = data.Name
				} else {
					defaultValue += "-" + data.Name
				}
			}
		}
		q := &survey.Input{
			Message: "Namespace:",
			Default: defaultValue,
			Help: "Th kubernetes namespace name to use for this Environment",
		}
		// TODO validate/transform to match valid kubnernetes names syntax
		err := survey.AskOne(q, &data.Spec.Namespace, survey.Required)
		if err != nil {
			return err
		}
	}
	if config.Spec.Cluster != "" {
		data.Spec.Cluster = config.Spec.Cluster
	} else {
		// lets not show the UI for this if users specify the namespace via arguments
		if !createMode || config.Spec.Namespace == "" {
			defaultValue := data.Spec.Cluster
			q := &survey.Input{
				Message: "Cluster URL:",
				Default: defaultValue,
				Help:    "The kubernetes cluster URL to use to host this Environment",
			}
			// TODO validate/transform to match valid kubnernetes cluster syntax
			err := survey.AskOne(q, &data.Spec.Cluster, nil)
			if err != nil {
				return err
			}
		}
	}
	if string(config.Spec.PromotionStrategy) != "" {
		data.Spec.PromotionStrategy = config.Spec.PromotionStrategy
	} else {
		// TODO edit the promotion strategy
	}
	if string(data.Spec.PromotionStrategy) == "" {
		data.Spec.PromotionStrategy = PromotionStrategyTypeAutomatic
	}

	return nil
}