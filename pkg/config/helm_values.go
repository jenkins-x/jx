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

type HelmValuesConfig struct {
	ExposeController *ExposeController `yaml:"expose,omitempty"`
}

type HelmValuesConfigService struct {
	FileName string
	Config   HelmValuesConfig
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
