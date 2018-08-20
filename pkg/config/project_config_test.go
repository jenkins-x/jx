package config

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tests"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
)

func TestProjectConfigMarshal(t *testing.T) {
	t.Parallel()
	projectConfig := &ProjectConfig{
		Builds: []*BranchBuild{
			{
				Kind: "release",
				Build: Build{
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

	data, err := yaml.Marshal(projectConfig)
	assert.NoError(t, err)

	if tests.IsDebugLog() {
		text := string(data)
		log.Infof("Generated YAML: %s\n", text)
	}

	copy := &ProjectConfig{}

	err = yaml.Unmarshal(data, copy)
	assert.NoError(t, err)

	assert.True(t, projectConfig.Builds[0].ExcludePodTemplateEnv)
	assert.True(t, projectConfig.Builds[0].ExcludePodTemplateVolumes)
}
