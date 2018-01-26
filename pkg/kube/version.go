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

// GetName returns the app name
func GetName(r *metav1.ObjectMeta) string {
	if r != nil {
		name := r.Name
		ns := r.Namespace
		if ns != "" {
			// for helm deployments which prefix the namespace in the name lets strip it
			prefix := ns + "-"
			if strings.HasPrefix(name, prefix) {
				return strings.TrimPrefix(name, prefix)
			}
		}
		labels := r.Labels
		if labels != nil {
			v := labels["app"]
			if v != "" {
				return v
			}
		}
		return name
	}
	return ""
}

// GetCommitSha returns the git commit sha
func GetCommitSha(r *metav1.ObjectMeta) string {
	if r != nil {
		annotations := r.Annotations
		if annotations != nil {
			return annotations["jenkins.io/git-sha"]
		}
	}
	return ""
}

// GetCommitURL returns the git commit URL
func GetCommitURL(r *metav1.ObjectMeta) string {
	if r != nil {
		annotations := r.Annotations
		if annotations != nil {
			return annotations["jenkins.io/git-url"]
		}
	}
	return ""
}
