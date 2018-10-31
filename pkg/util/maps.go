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
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}
