package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// ComplianceCheck represents the compliance checks performed for a particular pipeline run
type ComplianceCheck struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ComplianceCheckSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComnplianceCheckList is a list of ComplianceChecks
type ComplianceCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ComplianceCheck `json:"items"`
}

// ComplianceCheckSpec provides details of a particular Compliance Check
type ComplianceCheckSpec struct {
	PipelineActivity ResourceReference     `json:"pipelineActivity,omitempty"  protobuf:"bytes,1,opt,name=pipelineActivity"`
	Checks           []ComplianceCheckItem `json:"checks,omitempty"  protobuf:"bytes,2,opt,name=checks"`
	Checked          bool                  `json:"checked,omitempty"  protobuf:"bytes,3,opt,name=checked"`
}

type ComplianceCheckItem struct {
	Name        string `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Description string `json:"description,omitempty"  protobuf:"bytes,2,opt,name=description"`
	Pass        bool   `json:"pass,omitempty"  protobuf:"bytes,3,opt,name=pass"`
}
