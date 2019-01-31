package kube

import (
	corev1 "k8s.io/api/core/v1"
)

// GetSliceEnvVar returns the EnvVar for the given name or nil if none exists in the slice
func GetSliceEnvVar(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, envVar := range envVars {
		if envVar.Name == name {
			copy := envVar
			return &copy
		}
	}
	return nil
}
