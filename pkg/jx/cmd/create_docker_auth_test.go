package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestDockerAuth(t *testing.T) {
	t.Parallel()
	o := cmd.CreateDockerAuthOptions{
		Host:   "angoothachap.private.docker.registry",
		User:   "angoothachap",
		Secret: "AngoothachapDockerHubToken",
	}

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		nil,
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""))

	err := o.Run()
	assert.NoError(t, err)
}
