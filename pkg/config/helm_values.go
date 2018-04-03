package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type ExposeControllerConfig struct {
	Domain  string `yaml:"domain,omitempty"`
	Exposer string `yaml:"exposer"`
	HTTP    string `yaml:"http"`
	TLSAcme string `yaml:"tlsacme"`
}
type ExposeController struct {
	Config      ExposeControllerConfig `yaml:"config,omitempty"`
	Annotations map[string]string      `yaml:"Annotations,omitempty"`
}

type JenkinsValuesConfig struct {
	Servers JenkinsServersValuesConfig `yaml:"Servers,omitempty"`
}

type JenkinsServersValuesConfig struct {
	Gitea []JenkinsGiteaServersValuesConfig  `yaml:"Gitea,omitempty"`
	GHE   []JenkinsGithubServersValuesConfig `yaml:"GHE,omitempty"`
}

type JenkinsGiteaServersValuesConfig struct {
	Name       string `yaml:"Name,omitempty"`
	Url        string `yaml:"Url,omitempty"`
	Credential string `yaml:"Credential,omitempty"`
}

type JenkinsGithubServersValuesConfig struct {
	Name string `yaml:"Name,omitempty"`
	Url  string `yaml:"Url,omitempty"`
}

type HelmValuesConfig struct {
	ExposeController *ExposeController   `yaml:"expose,omitempty"`
	Jenkins          JenkinsValuesConfig `yaml:"jenkins,omitempty"`
}

type HelmValuesConfigService struct {
	FileName string
	Config   HelmValuesConfig
}

// GetOrCreateFirstGitea returns the first gitea server creating one if required
func (c *JenkinsServersValuesConfig) GetOrCreateFirstGitea() *JenkinsGiteaServersValuesConfig {
	if len(c.Gitea) == 0 {
		c.Gitea = []JenkinsGiteaServersValuesConfig{
			{
				Name:       "gitea",
				Credential: "jenkins-x-git",
			},
		}
	}
	return &c.Gitea[0]
}

func (c *HelmValuesConfig) AddExposeControllerValues(cmd *cobra.Command, ignoreDomain bool) {
	if !ignoreDomain {
		cmd.Flags().StringVarP(&c.ExposeController.Config.Domain, "domain", "", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	}
	keepJob := false

	cmd.Flags().StringVarP(&c.ExposeController.Config.HTTP, "http", "", "true", "Toggle creating http or https ingress rules")
	cmd.Flags().StringVarP(&c.ExposeController.Config.Exposer, "exposer", "", "Ingress", "Used to describe which strategy exposecontroller should use to access applications")
	cmd.Flags().StringVarP(&c.ExposeController.Config.TLSAcme, "tls-acme", "", "false", "Used to enable automatic TLS for ingress")
	cmd.Flags().BoolVarP(&keepJob, "keep-exposecontroller-job", "", false, "Prevents Helm deleting the exposecontroller Job and Pod after running.  Useful for debugging exposecontroller logs but you will need to manually delete the job if you update an environment")

	annotations := make(map[string]string)
	annotations["helm.sh/hook"] = "post-install,post-upgrade"
	if !keepJob {
		annotations["helm.sh/hook-delete-policy"] = "hook-succeeded"
	}
	c.ExposeController.Annotations = annotations
}

func (c HelmValuesConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshall helm values %v", err)
	}
	return string(b), nil
}
