package v1alpha1_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/syntax/syntax.jenkins.io/v1alpha1"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
)

func TestProjectConfigMarshal(t *testing.T) {
	t.Parallel()
	projectConfig := &v1alpha1.ProjectConfig{
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
		PipelineConfig: &v1alpha1.PipelineConfig{
			Pipelines: v1alpha1.Pipelines{
				PullRequest: &v1alpha1.PipelineLifecycles{
					Build: &v1alpha1.PipelineLifecycle{
						Steps: []*v1alpha1.PipelineStep{
							{
								Command: "mvn test",
							},
						},
					},
				},
				Release: &v1alpha1.PipelineLifecycles{
					Build: &v1alpha1.PipelineLifecycle{
						Steps: []*v1alpha1.PipelineStep{
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
		log.Infof("Generated YAML: %s\n", text)
	}

	copy := &v1alpha1.ProjectConfig{}

	err = yaml.Unmarshal(data, copy)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(projectConfig.Env), "len(projectConfig.Env)")
	assert.NotNil(t, projectConfig.PipelineConfig, "projectConfig.PipelineConfig")
	assert.NotNil(t, projectConfig.PipelineConfig.Pipelines.Release, "projectConfig.PipelineConfig.Pipelines.Release")
	assert.Equal(t, 1, len(projectConfig.PipelineConfig.Env), "len(projectConfig.PipelineConfig.Env)")
}
