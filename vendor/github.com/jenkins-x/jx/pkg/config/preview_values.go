package config

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type Image struct {
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty"`
}

type Preview struct {
	Image *Image `yaml:"image,omitempty"`
}

type PreviewValuesConfig struct {
	ExposeController *ExposeController `yaml:"expose,omitempty"`
	Preview          *Preview          `yaml:"preview,omitempty"`
}

func (c PreviewValuesConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshall helm preview values %v", err)
	}
	return string(b), nil
}
