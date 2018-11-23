package storage

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type SecretStore interface {
	// Write saves some secret data to the store
	Write(secretName string, bytes []byte) error

	// WriteObject writes a named object to the store
	WriteObject(secretName string, obj interface{}) error

	// Read reads some secret data from the store
	Read(secretName string) ([]byte, error)

	// ReadObject reads an object from the store
	ReadObject(s string, secret interface{}) error
}

//
type fileSecretStore struct {
}

// NewFileSecretStore creates a SecretStore that stores its data to the filesystem in YAML
func NewFileSecretStore() SecretStore {
	return &fileSecretStore{}
}

// Write writes a secret to the filesystem in YAML format
func (f *fileSecretStore) Write(fileName string, bytes []byte) error {
	return ioutil.WriteFile(fileName, bytes, util.DefaultWritePermissions)
}

// WriteObject writes a secret to the filesystem in YAML format
func (f *fileSecretStore) WriteObject(fileName string, obj interface{}) error {
	y, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrapf(err, "Unable to marshal object to yaml: %v", obj)
	}
	return f.Write(fileName, y)
}

func (f *fileSecretStore) Read(fileName string) ([]byte, error) {
	return ioutil.ReadFile(fileName)
}

// ReadObject reads an object from the filesystem as yaml
func (f *fileSecretStore) ReadObject(fileName string, secret interface{}) error {
	data, err := f.Read(fileName)
	if err != nil {
		return errors.Wrapf(err, "Unable to read %s", fileName)
	}
	return yaml.Unmarshal(data, secret)
}
