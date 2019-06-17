package config_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
)

func TestProjectConfigMarshal(t *testing.T) {
	t.Parallel()
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
		PipelineConfig: &jenkinsfile.PipelineConfig{
			Pipelines: jenkinsfile.Pipelines{
				PullRequest: &jenkinsfile.PipelineLifecycles{
					Build: &jenkinsfile.PipelineLifecycle{
						Steps: []*syntax.Step{
							{
								Command: "mvn test",
							},
						},
					},
				},
				Release: &jenkinsfile.PipelineLifecycles{
					Build: &jenkinsfile.PipelineLifecycle{
						Steps: []*syntax.Step{
							{
								Command: "mvn test",
							},
							{
								Command: "mvn deploy",
							},
							{
								Command: "jx promote --all-auto",
							},
						},
					},
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "PREVIEW_VERSION",
					Value: "0.0.0-SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER",
				},
			},
		},
	}

	data, err := yaml.Marshal(projectConfig)
	assert.NoError(t, err)

	if tests.IsDebugLog() {
		text := string(data)
		log.Logger().Infof("Generated YAML: %s", text)
	}

	copy := &config.ProjectConfig{}

	err = yaml.Unmarshal(data, copy)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(projectConfig.Env), "len(projectConfig.Env)")
	assert.NotNil(t, projectConfig.PipelineConfig, "projectConfig.PipelineConfig")
	assert.NotNil(t, projectConfig.PipelineConfig.Pipelines.Release, "projectConfig.PipelineConfig.Pipelines.Release")
	assert.Equal(t, 1, len(projectConfig.PipelineConfig.Env), "len(projectConfig.PipelineConfig.Env)")
}
