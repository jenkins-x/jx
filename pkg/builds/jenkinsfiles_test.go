package builds

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsfileGenerator(t *testing.T) {
	projectConfig := &config.ProjectConfig{
		BuildPack: "maven",
		Env: []corev1.EnvVar{
			{
				Name:  "ORG",
				Value: "myorg",
			},
			{
				Name:  "APP_NAME",
				Value: "thingy",
			},
		},
		Builds: []*config.BranchBuild{
			{
				Kind: "pullRequest",
				Name: "Pull Request Pipeline",
				Build: config.Build{
					Steps: []corev1.Container{
						{
							Args: []string{"mvn", "test"},
						},
						{
							Args: []string{"mvn", "deploy"},
						},
						{
							Args: []string{"jx", "promote", "--all-auto"},
						},
					},
				},
				ExcludePodTemplateEnv:     true,
				ExcludePodTemplateVolumes: true,
				Env: []corev1.EnvVar{
					{
						Name:  "PREVIEW_VERSION",
						Value: "0.0.0-SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER",
					},
				},
			},
			{
				Kind: "release",
				Name: "Release Pipeline",
				Build: config.Build{
					Steps: []corev1.Container{
						{
							Args: []string{"mvn", "test"},
						},
					},
				},
				ExcludePodTemplateEnv:     true,
				ExcludePodTemplateVolumes: true,
			},
		},
	}

	text, err := NewJenkinsConverter(projectConfig).ToJenkinsfile()
	assert.NoError(t, err)

	if tests.IsDebugLog() {
		log.Infof("Generated: %s\n", text)
	}
}
