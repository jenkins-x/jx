package metapipeline

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tekton"
)

// Client defines the interface for meta pipeline creation and application.
type Client interface {
	// Create creates the Tekton CRDs needed for executing the pipeline as defined by the input parameters
	Create(pullRef PullRef, pipelineType PipelineKind, context string, envs map[string]string, labels map[string]string) (kube.PromoteStepActivityKey, tekton.CRDWrapper, error)

	// Apply takes the given CRDs for processing, usually applying them to the cluster.
	Apply(pipelineActivity kube.PromoteStepActivityKey, crds tekton.CRDWrapper) error

	// Close cleans up the resources use by this client.
	Close() error

	//DefaultImage returns the default image used for pipeline tasks if no other image is specified.
	DefaultImage() string

	// ServiceAccount returns the service account under which to execute the pipeline.
	ServiceAccount() string
}
