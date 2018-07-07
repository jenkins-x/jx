package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"
)

type DockerRegistryConfig struct {
	DockerRegistry DockerRegistry `yaml:"docker-registry,omitempty"`
}

type DockerRegistry struct {
	Prefix string `yaml:"prefix,omitempty"`
}

type DockerRegistryService struct {
	DockerRegistry DockerRegistryConfig
	Flags          DockerRegistryFlags
}

type DockerRegistryFlags struct {
	Prefix string
}

func (c DockerRegistryConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshall docker registry %v", err)
	}

	return string(b), nil
}

func (d *DockerRegistryService) AddDockerRegistryValues(cmd *cobra.Command) error {
	cmd.Flags().StringVarP(&d.Flags.Prefix, "docker-registry-prefix", "", "", "You can specifed a docker registry url")
	return nil
}

func (d *DockerRegistryService) NewDockerRegistryConfig() error {
	d.DockerRegistry = DockerRegistryConfig{
		DockerRegistry: DockerRegistry{
			Prefix: d.Flags.Prefix,
		},
	}

	return nil
}

func (d *DockerRegistryService) HasPrefix() bool {
	return d.DockerRegistry.DockerRegistry.Prefix != ""
}

func (d *DockerRegistryService) OverrideImages(makefileDir string) (err error) {
	if !d.HasPrefix() {
		return
	}

	prefix := d.DockerRegistry.DockerRegistry.Prefix

	type JenkinsMaster struct {
		Image string `yaml:"Image,omitempty"`
	}

	type Jenkins struct {
		Master JenkinsMaster `yaml:"Master,omitempty"`
	}

	type Values struct {
		Jenkins Jenkins `yaml:"jenkins,omitempty"`
	}

	values := Values{
		Jenkins: Jenkins{
			Master: JenkinsMaster{
				Image: prefix + "jenkinsxio/jenkinsx",
			},
		},
	}

	v, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	valuesFile := filepath.Join(makefileDir, "myvalues.yaml")
	err = ioutil.WriteFile(valuesFile, []byte(v), 0644)
	if err != nil {
		return err
	}

	return
}
