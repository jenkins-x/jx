package io

import (
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/ghodss/yaml"
)

// ConfigStore provides an interface for storing configs
type ConfigStore interface {
	// Write saves some secret data to the store
	Write(name string, bytes []byte) error

	// WriteObject writes a named object to the store
	WriteObject(name string, obj interface{}) error

	// Read reads some secret data from the store
	Read(name string) ([]byte, error)

	// ReadObject reads an object from the store
	ReadObject(name string, object interface{}) error
}

type fileStore struct {
}

// NewFileStore creates a ConfigStore that stores its data to the filesystem in YAML
func NewFileStore() ConfigStore {
	return &fileStore{}
}

// Write writes a secret to the filesystem in YAML format
func (f *fileStore) Write(fileName string, bytes []byte) error {
	return ioutil.WriteFile(fileName, bytes, util.DefaultWritePermissions)
}

// WriteObject writes a secret to the filesystem in YAML format
func (f *fileStore) WriteObject(fileName string, obj interface{}) error {
	y, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrapf(err, "Unable to marshal object to yaml: %v", obj)
	}
	return f.Write(fileName, y)
}

func (f *fileStore) Read(fileName string) ([]byte, error) {
	return ioutil.ReadFile(fileName)
}

// ReadObject reads an object from the filesystem as yaml
func (f *fileStore) ReadObject(fileName string, object interface{}) error {
	data, err := f.Read(fileName)
	if err != nil {
		return errors.Wrapf(err, "Unable to read %s", fileName)
	}
	return yaml.Unmarshal(data, object)
}
