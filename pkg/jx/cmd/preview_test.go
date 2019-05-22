package cmd_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	gits_test "github.com/jenkins-x/jx/pkg/gits/mocks"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
)

// Constants for some test data to be used.
const (
	namespace = "jx"
)

func TestGetPreviewValuesConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		opts               cmd.PreviewOptions
		env                map[string]string
		domain             string
		expectedYAMLConfig string
	}{
		{
			opts: cmd.PreviewOptions{
				HelmValuesConfig: config.HelmValuesConfig{
					ExposeController: &config.ExposeController{},
				},
			},
			env: map[string]string{
				cmd.DOCKER_REGISTRY: "my.registry",
				cmd.ORG:             "my-org",
				cmd.APP_NAME:        "my-app",
				cmd.PREVIEW_VERSION: "1.0.0",
			},
			expectedYAMLConfig: `expose:
  config: {}
preview:
  image:
    repository: my.registry/my-org/my-app
    tag: 1.0.0
`,
		},
		{
			opts: cmd.PreviewOptions{
				HelmValuesConfig: config.HelmValuesConfig{
					ExposeController: &config.ExposeController{
						Config: config.ExposeControllerConfig{
							HTTP:    "false",
							TLSAcme: "true",
						},
					},
				},
			},
			env: map[string]string{
				cmd.DOCKER_REGISTRY: "my.registry",
				cmd.ORG:             "my-org",
				cmd.APP_NAME:        "my-app",
				cmd.PREVIEW_VERSION: "1.0.0",
			},
			domain: "jenkinsx.io",
			expectedYAMLConfig: `expose:
  config:
    domain: jenkinsx.io
    http: "false"
    tlsacme: "true"
preview:
  image:
    repository: my.registry/my-org/my-app
    tag: 1.0.0
`,
		},
	}
	co := &opts.CommonOptions{}
	cmd.ConfigureTestOptions(co, gits_test.NewMockGitter(), helm_test.NewMockHelmer())

	for i, test := range tests {
		for k, v := range test.env {
			os.Setenv(k, v)
		}

		test.opts.CommonOptions = co
		config, err := test.opts.GetPreviewValuesConfig(nil, test.domain)
		if err != nil {
			t.Errorf("[%d] got unexpected err: %v", i, err)
			continue
		}

		configYAML, err := config.String()
		if err != nil {
			t.Errorf("[%d] %v", i, err)
			continue
		}

		if test.expectedYAMLConfig != configYAML {
			t.Errorf("[%d] expected %#v but got %#v", i, test.expectedYAMLConfig, configYAML)
		}
	}
}
