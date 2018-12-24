package io

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"
)

// ConfigStore provides an interface for storing configs
type ConfigStore interface {
	// Write saves some secret data to the store
	Write(name string, bytes []byte) error

	// WriteObject writes a named object to the store
	WriteObject(name string, object interface{}) error

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
func (f *fileStore) WriteObject(fileName string, object interface{}) error {
	y, err := yaml.Marshal(object)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal object to yaml: %v", object)
	}
	return f.Write(fileName, y)
}

// Read reads a secret form the filesystem
func (f *fileStore) Read(fileName string) ([]byte, error) {
	return ioutil.ReadFile(fileName)
}

// ReadObject reads an object from the filesystem as yaml
func (f *fileStore) ReadObject(fileName string, object interface{}) error {
	data, err := f.Read(fileName)
	if err != nil {
		return errors.Wrapf(err, "unable to read %s", fileName)
	}
	return yaml.Unmarshal(data, object)
}

type vaultStore struct {
	client vault.Client
	path   string
}

// NewVaultStore creates a new store which stores its data in Vault
func NewVaultStore(client vault.Client, path string) ConfigStore {
	return &vaultStore{
		client: client,
		path:   path,
	}
}

// Write store a secret in vault as an array of bytes
func (v *vaultStore) Write(name string, bytes []byte) error {
	data := map[string]interface{}{
		"data": bytes,
	}
	_, err := v.client.Write(v.secretPath(name), data)
	if err != nil {
		return errors.Wrapf(err, "unable to write data for secret '%s' to vault", name)
	}
	return nil
}

// Read reads a secret from vault wich was stored as an array of bytes
func (v *vaultStore) Read(name string) ([]byte, error) {
	secret, err := v.client.Read(v.secretPath(name))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read '%s' secret from vault", name)
	}
	data, ok := secret["data"]
	if !ok {
		return nil, fmt.Errorf("data not found for secret '%s'", name)
	}

	bytes, ok := data.([]byte)
	if !ok {
		return nil, fmt.Errorf("unable to convert the secret content '%s' to bytes", name)
	}

	return bytes, nil
}

// WriteObject writes a generic named object to vault
func (v *vaultStore) WriteObject(name string, object interface{}) error {
	_, err := v.client.WriteObject(v.secretPath(name), object)
	if err != nil {
		return errors.Wrapf(err, "undable to write the '%s' secret object to vault", name)
	}
	return nil
}

func (v *vaultStore) ReadObject(name string, object interface{}) error {
	err := v.client.ReadObject(v.secretPath(name), object)
	if err != nil {
		return errors.Wrapf(err, "undable to read the '%s' secret object from vault", name)
	}
	return nil
}

func (v *vaultStore) secretPath(name string) string {
	return v.path + name
}
