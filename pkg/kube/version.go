package kube

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetVersion returns the version from the labels on the deployment if it can be deduced
func GetVersion(r *metav1.ObjectMeta) string {
	if r != nil {
		labels := r.Labels
		if labels != nil {
			v := labels["version"]
			if v != "" {
				return v
			}
			v = labels["chart"]
			if v != "" {
				arr := strings.Split(v, "-")
				last := arr[len(arr)-1]
				if last != "" {
					return last
				}
				return v
			}
		}
	}
	return ""
}
