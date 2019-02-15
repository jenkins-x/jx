package flagger

import (
	"k8s.io/api/apps/v1beta1"
)

// IsCanaryAuxiliaryDeployment returns whether this deployment has been created automatically by Flagger from a Canary object
func IsCanaryAuxiliaryDeployment(d v1beta1.Deployment) bool {
	ownerReferences := d.GetObjectMeta().GetOwnerReferences()
	for i := range ownerReferences {
		if ownerReferences[i].Kind == "Canary" {
			return true
		}
	}
	return false
}
