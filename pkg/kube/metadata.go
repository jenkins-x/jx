package kube

import "strconv"

// MergeMaps merges all the maps together with the entries in the last map overwriting any earlier values
//
// so if you want to add some annotations to a resource you can do
// resource.Annotations = kube.MergeMaps(resource.Annotations, myAnnotations)
func MergeMaps(maps ...map[string]string) map[string]string {
	answer := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			answer[k] = v
		}
	}
	return answer
}

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
