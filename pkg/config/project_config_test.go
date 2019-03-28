package config_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
)

func TestProjectConfigMarshal(t *testing.T) {
	t.Parallel()
	projectConfig := &config.ProjectConfig{
		Builds: []*config.BranchBuild{
			{
				Kind: "release",
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

	data, err := yaml.Marshal(projectConfig)
	assert.NoError(t, err)

	if tests.IsDebugLog() {
		text := string(data)
		log.Infof("Generated YAML: %s\n", text)
	}

	copy := &config.ProjectConfig{}

	err = yaml.Unmarshal(data, copy)
	assert.NoError(t, err)

	assert.True(t, projectConfig.Builds[0].ExcludePodTemplateEnv)
	assert.True(t, projectConfig.Builds[0].ExcludePodTemplateVolumes)
}
