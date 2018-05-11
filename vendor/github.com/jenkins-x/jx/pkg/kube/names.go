package kube

import (
	"bytes"
	"strings"
)

// ToValidName converts the given string into a valid kubernetes resource name
func ToValidName(name string) string {
	return toValidName(name, false)
}

// ToValidNameWithDots converts the given string into a valid kubernetes resource name
func ToValidNameWithDots(name string) string {
	return toValidName(name, true)
}

func toValidName(name string, allowDots bool) string {
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
			if !allowDots && ch == '.' {
				ch = '-'
			}
			if !(ch >= 'a' && ch <= 'z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '.' {
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
