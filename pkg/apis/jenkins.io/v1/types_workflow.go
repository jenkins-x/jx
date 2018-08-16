package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Workflow represents pipeline activity for a particular run of a pipeline
type Workflow struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   WorkflowSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status WorkflowStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// WorkflowSpec is the specification of the pipeline activity
type WorkflowSpec struct {
	PipelineName string         `json:"pipeline,omitempty" protobuf:"bytes,1,opt,name=pipeline"`
	Steps        []WorkflowStep `json:"steps,omitempty" protobuf:"bytes,7,opt,name=steps"`
}

// WorkflowStep represents a step in a pipeline activity
type WorkflowStep struct {
	Kind          WorkflowStepKindType  `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	Name          string                `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Description   string                `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`
	Preconditions WorkflowPreconditions `json:"trigger,omitempty" protobuf:"bytes,3,opt,name=trigger"`
	Promote       *PromoteWorkflowStep  `json:"promote,omitempty" protobuf:"bytes,4,opt,name=promote"`
}

// PromoteWorkflowStep is the step of promoting a version of an application to an environment
type PromoteWorkflowStep struct {
	Environment string `json:"environment,omitempty" protobuf:"bytes,1,opt,name=environment"`
}

// WorkflowPreconditions is the trigger to start a step
type WorkflowPreconditions struct {
	// the names of the environments which need to have promoted before this step can be triggered
	Environments []string `json:"environments,omitempty" protobuf:"bytes,1,opt,name=environments"`
}

// WorkflowStatus is the status for an Environment resource
type WorkflowStatus struct {
	Version string `json:"version,omitempty"  protobuf:"bytes,1,opt,name=version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkflowList is a list of Workflow resources
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Workflow `json:"items"`
}

// WorkflowStepKindType is a kind of step
type WorkflowStepKindType string

const (
	// WorkflowStepKindTypeNone no kind yet
	WorkflowStepKindTypeNone WorkflowStepKindType = ""
	// WorkflowStepKindTypePromote a promote activity
	WorkflowStepKindTypePromote WorkflowStepKindType = "Promote"
)

// WorkflowStatusType is the status of an activity; usually succeeded or failed/error on completion
type WorkflowStatusType string

const (
	// WorkflowStatusTypeNone an activity step has not started yet
	WorkflowStatusTypeNone WorkflowStatusType = ""
	// WorkflowStatusTypePending an activity step is waiting to start
	WorkflowStatusTypePending WorkflowStatusType = "Pending"
	// WorkflowStatusTypeRunning an activity is running
	WorkflowStatusTypeRunning WorkflowStatusType = "Running"
	// WorkflowStatusTypeSucceeded an activity completed successfully
	WorkflowStatusTypeSucceeded WorkflowStatusType = "Succeeded"
	// WorkflowStatusTypeFailed an activity failed
	WorkflowStatusTypeFailed WorkflowStatusType = "Failed"
	// WorkflowStatusTypeWaitingForApproval an activity is waiting for approval
	WorkflowStatusTypeWaitingForApproval WorkflowStatusType = "WaitingForApproval"
	// WorkflowStatusTypeError there is some error with an activity
	WorkflowStatusTypeError WorkflowStatusType = "Error"
)

func (s WorkflowStatusType) String() string {
	return string(s)
}
