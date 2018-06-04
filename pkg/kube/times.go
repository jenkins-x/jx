package kube

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToMetaTime converts the go time pointer to a meta time
func ToMetaTime(t *time.Time) *metav1.Time {
	if t == nil {
		return nil
	}
	return &metav1.Time{
		Time: *t,
	}
}
