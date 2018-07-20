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
