package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type ExposeController struct {
	Domain  string `yaml:"domain,omitempty"`
	Exposer string `yaml:"exposer"`
	HTTP    bool   `yaml:"http"`
	TLSAcme bool   `yaml:"tlsacme"`
}

type HelmValuesConfig struct {
	ExposeController *ExposeController `yaml:"exposecontroller,omitempty"`
}

type HelmValuesConfigService struct {
	FileName string
	Config   HelmValuesConfig
}

func (c *HelmValuesConfig) AddExposeControllerValues(cmd *cobra.Command, ignoreDomain bool) {
	if !ignoreDomain {
		cmd.Flags().StringVarP(&c.ExposeController.Domain, "domain", "", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	}
	cmd.Flags().BoolVarP(&c.ExposeController.HTTP, "http", "", true, "Toggle creating http or https ingress rules")
	cmd.Flags().StringVarP(&c.ExposeController.Exposer, "exposer", "", "Ingress", "Used to describe which strategy exposecontroller should use to access applications")
	cmd.Flags().BoolVarP(&c.ExposeController.TLSAcme, "tls-acme", "", false, "Used to enable automatic TLS for ingress")
}

func (c HelmValuesConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshall helm values %v", err)
	}
	return string(b), nil
}
