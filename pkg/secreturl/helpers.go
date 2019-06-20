package secreturl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// ReplaceURIs will replace any URIs with the given regular expression and scheme using the secret URL client
func ReplaceURIs(s string, client Client, r *regexp.Regexp, schemePrefix string) (string, error) {
	if !strings.HasSuffix(schemePrefix, ":") {
		return s, fmt.Errorf("the scheme prefix should end with ':' but was %s", schemePrefix)
	}
	var err error
	answer := r.ReplaceAllStringFunc(s, func(found string) string {
		// Stop once we have an error
		if err == nil {
			pathAndKey := strings.Trim(strings.TrimPrefix(found, schemePrefix), "\"")
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
