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
		if m != nil {
			for k, v := range m {
				answer[k] = v
			}
		}
	}
	return answer
}

// CombineMapTrees recursively copies all the values from the input map into the destination map preserving any missing entries in the destination
func CombineMapTrees(destination map[string]interface{}, input map[string]interface{}) {
	for k, v := range input {
		old, exists := destination[k]
		if exists {
			vm, ok := v.(map[string]interface{})
			if ok {
				oldm, ok := old.(map[string]interface{})
				if ok {
					// if both entries are maps lets combine them
					// otherwise we assume that the input entry is correct
					CombineMapTrees(oldm, vm)
					continue
				}
			}
		}
		destination[k] = v
	}
}
