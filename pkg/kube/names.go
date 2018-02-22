package kube

import (
	"bytes"
	"strings"
)

// ToValidName converts the given string into a valid kubernetes resource name
func ToValidName(name string) string {
	var buffer bytes.Buffer
	first := true
	lastCharDash := false
	lower := strings.ToLower(name)
	for _, ch := range lower {
		if first {
			// strip non letters at start
			if ch >= 'a' && ch <= 'z' {
				buffer.WriteRune(ch)
				first = false
			}
		} else {
			if !(ch >= 'a' && ch <= 'z') && !(ch >= '0' && ch <= '9') && ch != '-' {
				ch = '-'
			}

			if ch != '-' || !lastCharDash {
				buffer.WriteRune(ch)
			}
			lastCharDash = ch == '-'
		}
	}
	answer := buffer.String()
	for {
		if strings.HasSuffix(answer, "-") {
			answer = strings.TrimSuffix(answer, "-")
		} else {
			break
		}
	}
	return answer
}
