package vault

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

const (
	usernameKey = "Username"
	passwordKey = "Password"
)

var vaultURIRegex = regexp.MustCompile(`vault:[-_\w\/:]*`)

// WriteYAMLFiles stores the given YAML files in vault. The final secret path is
// a concatenation of the 'path' with the file name.
func WriteYamlFiles(client Client, path string, files ...string) error {
	secrets := map[string]string{}
	for _, file := range files {
		exists, err := util.FileExists(file)
		if exists && err == nil {
			empty, err := util.FileIsEmpty(file)
			if !empty && err == nil {
				content, err := ioutil.ReadFile(file)
				if err != nil {
					return errors.Wrapf(err, "reading secrets file '%s'", file)
				}
				key := filepath.Base(file)
				secrets[key] = string(content)
			}
		}
	}
	for secretName, secret := range secrets {
		secretPath := path + secretName
		_, err := client.WriteYaml(secretPath, secret)
		if err != nil {
			return errors.Wrapf(err, "storing the YAML file '%s' into vault", secretName)
		}
	}
	return nil
}

// WriteBasicAuth stores the basic authentication credentials in vault at the given path.
func WriteBasicAuth(client Client, path string, auth config.BasicAuth) error {
	return WriteMap(client, path, map[string]interface{}{
		usernameKey: auth.Username,
		passwordKey: auth.Password,
	})
}

// WriteMap stores the map in vault at the given path.
func WriteMap(client Client, path string, secret map[string]interface{}) error {
	_, err := client.Write(path, secret)
	if err != nil {
		return errors.Wrapf(err, "storing basic auth credentials into vault at path '%s'", path)
	}
	return nil
}

// ToURI constructs a vault: URI for the given path and key
func ToURI(path string, key string) string {
	return fmt.Sprintf("vault:%s:%s", path, key)
}

// ReplaceURIs will replace any vault: URIs in a string, using the vault client
func ReplaceURIs(s string, client Client) (string, error) {
	var err error
	answer := vaultURIRegex.ReplaceAllStringFunc(s, func(found string) string {
		// Stop once we have an error
		if err == nil {
			pathAndKey := strings.Trim(strings.TrimPrefix(found, "vault:"), "\"")
			parts := strings.Split(pathAndKey, ":")
			if len(parts) != 2 {
				err = errors.Errorf("cannot parse %s as path:key", pathAndKey)
				return ""
			}
			secret, err1 := client.Read(parts[0])
			if err1 != nil {
				err = errors.Wrapf(err1, "reading %s from vault", parts[0])
				return ""
			}
			if v, ok := secret[parts[1]]; !ok {
				err = errors.Errorf("unable to find %s in secret at %s", parts[1], parts[0])
				return ""
			} else {
				result, err1 := util.AsString(v)
				if err1 != nil {
					err = errors.Wrapf(err1, "converting %v to string", v)
					return ""
				}
				return result
			}
		}
		return found
	})
	if err != nil {
		return "", errors.Wrapf(err, "replacing vault paths in %s", s)
	}
	return answer, nil
}
