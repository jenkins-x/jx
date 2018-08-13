// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/jenkins-x/jx/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Environments returns a EnvironmentInformer.
	Environments() EnvironmentInformer
	// EnvironmentRoleBindings returns a EnvironmentRoleBindingInformer.
	EnvironmentRoleBindings() EnvironmentRoleBindingInformer
	// GitServices returns a GitServiceInformer.
	GitServices() GitServiceInformer
	// PipelineActivities returns a PipelineActivityInformer.
	PipelineActivities() PipelineActivityInformer
	// Releases returns a ReleaseInformer.
	Releases() ReleaseInformer
	// Teams returns a TeamInformer.
	Teams() TeamInformer
	// Users returns a UserInformer.
	Users() UserInformer
	// Workflows returns a WorkflowInformer.
	Workflows() WorkflowInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Environments returns a EnvironmentInformer.
func (v *version) Environments() EnvironmentInformer {
	return &environmentInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// EnvironmentRoleBindings returns a EnvironmentRoleBindingInformer.
func (v *version) EnvironmentRoleBindings() EnvironmentRoleBindingInformer {
	return &environmentRoleBindingInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// GitServices returns a GitServiceInformer.
func (v *version) GitServices() GitServiceInformer {
	return &gitServiceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PipelineActivities returns a PipelineActivityInformer.
func (v *version) PipelineActivities() PipelineActivityInformer {
	return &pipelineActivityInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Releases returns a ReleaseInformer.
func (v *version) Releases() ReleaseInformer {
	return &releaseInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Teams returns a TeamInformer.
func (v *version) Teams() TeamInformer {
	return &teamInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Workflows returns a WorkflowInformer.
func (v *version) Workflows() WorkflowInformer {
	return &workflowInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
