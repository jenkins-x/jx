package util

import (
	"fmt"
	"strings"
)

// EncodeKubernetesName takes a string and turns it into a form suitable for use as a Kubernetes name.
// K8S names are lower case, numbers, '-', and '.'. Invalid characters are percent-encoded in a _similar_ way to URL
// encoding, with a percent followed by the numerical character code. Conversion is:
//   Upper case letters -> lower case.
//   'a-z', '0-9', '-', '.' -> left as-is.
//   Any other characters: percent-encoded ('%' + rune hex code).
func EncodeKubernetesName(name string) string {
	name = strings.ToLower(name)
	encodedName := strings.Builder{}

	for _, ch := range name {
		if allowedInK8SName(ch) {
			encodedName.WriteRune(ch)
		} else {
			encodedName.WriteString(fmt.Sprintf("%%%X", int(ch)))
		}

	}
	return encodedName.String()
}

func allowedInK8SName(ch rune) bool {
	return ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '.'
}
