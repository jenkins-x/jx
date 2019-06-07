package builds_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateBuildPodInfo(t *testing.T) {
	t.Parallel()

	pod1File := AssertLoadPod(t, filepath.Join("test_data", "pod1.yml"))
	if pod1File != nil {
		b := builds.CreateBuildPodInfo(pod1File)

		//log.Logger().Infof("Found build info %#v\n", b)

		assert.Equal(t, "jenkins-x-jenkins-x-serverless-PR-52-6", b.Name, "Name")
		assert.Equal(t, "jenkins-x", b.Organisation, "Organisation")
		assert.Equal(t, "57be1dc2-ddb4-11e8-8ea8-0a580a300275-h59fj", b.PodName, "PodName")
		assert.Equal(t, "jenkins-x-serverless", b.Repository, "Repository")
		assert.Equal(t, "PR-52", b.Branch, "Branch")
		assert.Equal(t, "6", b.Build, "Build")
		assert.Equal(t, "jenkins-x/jenkins-x-serverless/PR-52", b.Pipeline, "Pipeline")
		assert.Equal(t, "b662eb177fdd4252220399aa8da809411d87b8ed", b.LastCommitSHA, "LastCommitSHA")
		assert.Equal(t, "https://github.com/jenkins-x/jenkins-x-serverless.git", b.GitURL, "GitURL")
		assert.Equal(t, "jenkinsxio/jenkins-cwp:0.1.33", b.FirstStepImage, "FirstStepImage")
	}
}

func AssertLoadPod(t *testing.T, fileName string) *corev1.Pod {
	if tests.AssertFileExists(t, fileName) {
		pod := &corev1.Pod{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, pod)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return pod
			}

		}
	}
	return nil
}
