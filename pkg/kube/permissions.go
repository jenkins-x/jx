package kube

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	v1 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
)

// Resource is the representation of any Kubernetes resource
type Resource string

// Verb is the representation of the different verbs that can be checked for Kubernetes resources
type Verb string

const (
	// ClusterRoleBindings is the clusterrolebindings.rbac.authorization.k8s.io resource
	ClusterRoleBindings Resource = "clusterrolebindings"
	// ClusterRoles is the clusterroles.rbac.authorization.k8s.io resource
	ClusterRoles Resource = "clusterrole"
	// CustomResourceDefinitions is the customresourcedefinitions.apiextensions.k8s.io resource
	CustomResourceDefinitions Resource = "customresourcedefinitions"
	// All is the representation of '*' meaning all resources
	All Resource = "'*'"
	// Create represents the create verb
	Create Verb = "create"
	// Delete represents the delete verb
	Delete Verb = "delete"
	// Get represents the get verb
	Get Verb = "get"
	// List represents the list verb
	List Verb = "list"
	// Update represents the update verb
	Update Verb = "use"
	// Watch represents the watch verb
	Watch Verb = "watch"
)

// CanI will take a verb and a list of resources and it will check whether the current user / service account can
// perform that verb against the resources in the Kubernetes cluster
func CanI(kubeClient kubernetes.Interface, verb Verb, resources ...Resource) (bool, []error) {
	var errList []error
	for _, resource := range resources {
		result, err := kubeClient.AuthorizationV1().SelfSubjectAccessReviews().Create(&v1.SelfSubjectAccessReview{
			Spec: v1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &v1.ResourceAttributes{
					Verb:     string(verb),
					Resource: string(resource),
				},
			},
		})
		if err != nil {
			errList = append(errList, err)
		} else {
			if !result.Status.Allowed || result.Status.Denied {
				log.Logger().Debugf("Authentication evaluation denied due to: %s", result.Status.Reason)
				return false, errList
			}
		}
	}

	if len(errList) > 0 {
		return false, errList
	}
	return true, nil
}
