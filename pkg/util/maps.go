package util

// StringMapHasValue returns true if the given map contains the given value
func StringMapHasValue(m map[string]string, value string) bool {
	if m == nil {
		return false
	}
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

// MapKeys returns the keys of a given map
func MapKeys(m map[string]string) []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

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
