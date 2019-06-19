package localvault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/secreturl"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

var localURIRegex = regexp.MustCompile(`local:[-_\w\/:]*`)

// FileSystemClient a local file system based client loading/saving content from the given URL
type FileSystemClient struct {
	Dir string
}

// NewFileSystemClient create a new local file system based client loading content from the given URL
func NewFileSystemClient(dir string) secreturl.Client {
	return &FileSystemClient{
		Dir: dir,
	}
}

// Read reads a named secret from the vault
func (c *FileSystemClient) Read(secretName string) (map[string]interface{}, error) {
	name := c.fileName(secretName)
	exists, err := util.FileExists(name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if file exists %s", name)
	}
	if !exists {
		return nil, fmt.Errorf("local vault file does not exist: %s", name)
	}
	return helm.LoadValuesFile(name)
}

// ReadObject reads a generic named object from vault.
// The secret _must_ be serializable to JSON.
func (c *FileSystemClient) ReadObject(secretName string, secret interface{}) error {
	m, err := c.Read(secretName)
	if err != nil {
		return errors.Wrapf(err, "reading the secret %q from vault", secretName)
	}
	err = util.ToStructFromMapStringInterface(m, &secret)
	if err != nil {
		return errors.Wrapf(err, "deserializing the secret %q from vault", secretName)
	}
	return nil
}

// WriteObject writes a generic named object to the vault.
// The secret _must_ be serializable to JSON.
func (c *FileSystemClient) WriteObject(secretName string, secret interface{}) (map[string]interface{}, error) {
	path := c.fileName(secretName)
	dir, _ := filepath.Split(path)
	err := os.MkdirAll(dir, util.DefaultWritePermissions)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to ensure that parent directory exists %s", dir)
	}
	err = helm.SaveFile(path, secret)
	if err != nil {
		return nil, err
	}
	return c.Read(secretName)
}

func (c *FileSystemClient) ReplaceURIs(s string) (string, error) {
	return secreturl.ReplaceURIs(s, c, localURIRegex, "local:")
}

func (c *FileSystemClient) fileName(secretName string) string {
	return filepath.Join(c.Dir, secretName+".yaml")
}
