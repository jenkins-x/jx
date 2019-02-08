package kube

import (
	"bytes"
	"math"
	"strings"
)

// ToValidName converts the given string into a valid Kubernetes resource name
func ToValidName(name string) string {
	return toValidName(name, false, math.MaxInt32)
}

// ToValidNameWithDots converts the given string into a valid Kubernetes resource name
func ToValidNameWithDots(name string) string {
	return toValidName(name, true, math.MaxInt32)
}

// ToValidNameTruncated converts the given string into a valid Kubernetes resource name,
// truncating the result if it is more than maxLength characters.
func ToValidNameTruncated(name string, maxLength int) string {
	return toValidName(name, false, maxLength)
}

func toValidName(name string, allowDots bool, maxLength int) string {
	var buffer bytes.Buffer
	first := true
	lastCharDash := false
	lower := strings.ToLower(name)
	for _, ch := range lower {
		if buffer.Len()+1 > maxLength {
			break
		}
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

//EmailToK8sID converts the provided email address to a valid Kubernetes resource name, converting the @ to a .
func EmailToK8sID(email string) string {
	return ToValidNameWithDots(strings.Replace(email, "@", ".", -1))
}
