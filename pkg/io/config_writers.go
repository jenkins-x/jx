package io

import (
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultWritePermissions = 0640
)

//ConfigWriter interface for writing auth configuration
type ConfigWriter interface {
	Write(config *auth.AuthConfig) error
}

//FileConfigWriter file config write which keeps the path to the configuration file
type FileConfigWriter struct {
	filename string
}

//NewFileConfigWriter creates a new file config writer
func NewFileConfigWriter(filename string) *FileConfigWriter {
	return &FileConfigWriter{
		filename: filename,
	}
}

//Write writes the auth configuration into a file
func (f *FileConfigWriter) Write(config *auth.AuthConfig) error {
	if f.filename == "" {
		return errors.New("No config file name defined")
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshaling the config to yaml")
	}
	err = ioutil.WriteFile(f.filename, content, defaultWritePermissions)
	return nil
}
