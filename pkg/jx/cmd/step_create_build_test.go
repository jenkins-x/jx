package cmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	actualBuildFileName   = "build-release.yml"
	expectedBuildFileName = "expected-build-release.yml"

	MavenBuildPackYaml = `---
apiVersion: v1
kind: Pod
metadata:
  name: jenkins-maven
  labels:
    jenkins.io/kind: build-pod
spec:
  serviceAccount: jenkins
  nodeSelector:
  volumes:
  - name: workspace-volume
    emptyDir: {}
  - name: docker-daemon
    hostPath:
      path: /var/run/docker.sock
  - name: volume-0
    secret:
      secretName: jenkins-maven-settings
  - name: volume-1
    secret:
      secretName: jenkins-docker-cfg
  - name: volume-2
    secret:
      secretName: jenkins-release-gpg
  containers:
  - name: maven
    image: jenkinsxio/builder-maven:0.0.408
    args:
    - cat
    command:
    - /bin/sh
    - -c
    workingDir: /home/jenkins
    securityContext:
      privileged: true
    tty: true
    env:
    - name: DOCKER_REGISTRY
      valueFrom:
        configMapKeyRef:
          name: jenkins-x-docker-registry
          key: docker.registry
    - name: DOCKER_CONFIG
      value: /home/jenkins/.docker/
    - name: GIT_AUTHOR_EMAIL
      value: jenkins-x@googlegroups.com
    - name: GIT_AUTHOR_NAME
      value: jenkins-x-bot
    - name: GIT_COMMITTER_EMAIL
      value: jenkins-x@googlegroups.com
    - name: GIT_COMMITTER_NAME
      value: jenkins-x-bot
    - name: JENKINS_URL
      value: http://jenkins:8080
    - name: XDG_CONFIG_HOME
      value: /home/jenkins
    - name: _JAVA_OPTIONS
      value: -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap -Dsun.zip.disableMemoryMapping=true -XX:+UseParallelGC -XX:MinHeapFreeRatio=5 -XX:MaxHeapFreeRatio=10 -XX:GCTimeRatio=4 -XX:AdaptiveSizePolicyWeight=90 -Xms10m -Xmx192m
    resources:
      requests:
        cpu: 400m
        memory: 512Mi
      limits:
    volumeMounts:
      - mountPath: /home/jenkins
        name: workspace-volume
      - name: docker-daemon
        mountPath: /var/run/docker.sock
      - name: volume-0
        mountPath: /root/.m2/
      - name: volume-1
        mountPath: /home/jenkins/.docker
      - name: volume-2
        mountPath: /home/jenkins/.gnupg
`
)

func TestStepCreateBuild(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "test-step-create-build")
	assert.NoError(t, err)

	testData := path.Join("test_data", "step_create_build")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join(testData, name)
			testStepCreateBuild(t, tempDir, name, srcDir)
		}
	}
}

func testStepCreateBuild(t *testing.T, tempDir string, testcase string, srcDir string) error {
	testDir := filepath.Join(tempDir, testcase)
	util.CopyDir(srcDir, testDir, true)

	_, dirName := filepath.Split(testDir)

	k8sObjects := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.ConfigMapJenkinsPodTemplates,
				Namespace: "jx",
			},
			Data: map[string]string{
				"maven": MavenBuildPackYaml,
			},
		},
	}
	jxObjects := []runtime.Object{}

	o := &cmd.StepCreateBuildOptions{}
	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions, k8sObjects, jxObjects, gits.NewGitCLI(),
		nil, helm.NewHelmCLI("helm", helm.V2, dirName, true))
	o.Dir = testDir

	actualFile := filepath.Join(testDir, actualBuildFileName)
	expectedFile := filepath.Join(testDir, expectedBuildFileName)

	o.OutputDir = testDir

	err := o.Run()
	assert.NoError(t, err, "Failed with %s", err)
	if err == nil {
		err = tests.AssertEqualFileText(t, expectedFile, actualFile)
		if err != nil {
			return err
		}
	}
	return err
}
