package kube

import "strconv"

// IsResourceVersionNewer returns true if the first resource version is newer than the second
func IsResourceVersionNewer(v1 string, v2 string) bool {
	if v1 == v2 || v1 == "" {
		return false
	}
	if v2 == "" {
		return true
	}
	i1, e1 := strconv.Atoi(v1)
	i2, e2 := strconv.Atoi(v2)

	if e1 == nil && e2 != nil {
		return true
	}
	if e1 != nil {
		return false
	}
	return i1 > i2
}
