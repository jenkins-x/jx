package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	//"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"

	//"k8s.io/client-go/tools/clientcmd/api"

	"k8s.io/apimachinery/pkg/runtime"
	
)

func TestPromoteRun(t *testing.T) {

	// jx promote -b --all-auto --timeout 1h --version 1.0.0 --no-helm-update

	o := &cmd.PromoteOptions{
		Namespace:          "jx-testing",
		Environment:        "dev",
		Application:        "test_app",
		Pipeline:           "",
		Build:              "",
		Version:            "1.0.0", // --version 1.0.0
		ReleaseName:        "",
		LocalHelmRepoName:  "",
		HelmRepositoryURL:  "",
		NoHelmUpdate:       true, // --no-helm-update
		AllAutomatic:       true, // --all-auto
		NoMergePullRequest: false,
		NoPoll:             false,
		NoWaitAfterMerge:   false,
		IgnoreLocalFiles:   false,
		Timeout:            "1h", // --timeout 1h
		PullRequestPollTime:"20s",
		Filter:             "",
		Alias:              "",
		CommonOptions: cmd.CommonOptions{
			// Factory initialized by cmd.ConfigureTestOptionsWithResources
		},

		// test settings
		UseFakeHelm: true,
	}

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions, 
		[]runtime.Object{ // k8s objects 
		},
		[]runtime.Object{ // JX objects
			kube.NewPermanentEnvironment("staging"),
			kube.NewPermanentEnvironment("production"),
		},
		&gits.GitFake{}, helm_test.NewMockHelmer())

	err := o.Run()
	assert.NoError(t, err)


}