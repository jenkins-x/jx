package kube

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
