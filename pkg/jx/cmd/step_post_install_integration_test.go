// +build integration

package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jenkins/fake"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/testkube"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"testing"
)

func TestStepPostInstall(t *testing.T) {
	t.Parallel()

	dev := kube.CreateDefaultDevEnvironment("jx")
	testOrg := "mytestorg"
	testRepo := "mytestrepo"
	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrg+"/"+testRepo+".git")

	o := cmd.StepPostInstallOptions{
		StepOptions: cmd.StepOptions{
			CommonOptions: cmd.CommonOptions{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			},
		},
	}
	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kube.ConfigMapJenkinsX,
					Namespace: "jx",
				},
				Data: map[string]string{},
			},
			testkube.CreateFakeGitSecret(),
		},
		[]runtime.Object{
			dev,
			staging,
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, "", true),
	)

	o.BatchMode = true
	jenkinsClient := fake.NewFakeJenkins()
	o.SetJenkinsClient(jenkinsClient)
	o.GitClient = &gits.GitFake{}

	err := o.Run()
	require.NoError(t, err, "failed to run jx step post install")

	for _, job := range jenkinsClient.Jobs {
		t.Logf("Jenkins Job: %s at %s\n", job.Name, job.Url)
		for _, cj := range job.Jobs {
			t.Logf("\t child Job: %s at %s\n", cj.Name, cj.Url)
		}
	}

	// assert we have a jenkins job for the staging env repo
	job, err := jenkinsClient.GetJobByPath(testOrg, testRepo)
	require.NoError(t, err, "failed to query Jenkins Job for %s/%s", testOrg, testRepo)
	assert.Equal(t, job.Name, testRepo, "job.Name")

	t.Logf("Found Jenkins Job at URL: %s\n", job.Url)

	// TODO assert we have a webhook for the staging env repo

}
