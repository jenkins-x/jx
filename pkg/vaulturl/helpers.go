package vaulturl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

var vaultURIRegex = regexp.MustCompile(`vault:[-_\w\/:]*`)

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
			v, ok := secret[parts[1]]
			if !ok {
				err = errors.Errorf("unable to find %s in secret at %s", parts[1], parts[0])
				return ""
			}
			result, err1 := util.AsString(v)
			if err1 != nil {
				err = errors.Wrapf(err1, "converting %v to string", v)
				return ""
			}
			return result
		}
		return found
	})
	if err != nil {
		return "", errors.Wrapf(err, "replacing vault paths in %s", s)
	}
	return answer, nil
}

// ToURI constructs a vault: URI for the given path and key
func ToURI(path string, key string) string {
	return fmt.Sprintf("vault:%s:%s", path, key)
}
