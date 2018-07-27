package kube

import (
	corev1 "k8s.io/api/core/v1"
)

// GetEnvVar returns the env var if its defined for the given name
func GetEnvVar(container *corev1.Container, name string) *corev1.EnvVar {
	if container == nil {
		return nil
	}
	for i, _ := range container.Env {
		env := &container.Env[i]
		if env.Name == name {
			return env
		}
	}
	return nil
}
