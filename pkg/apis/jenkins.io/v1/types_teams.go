package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Team represents a request to create an actual Team which is a group of users, a development environment and optional other environments
type Team struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   TeamSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status TeamStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// TeamSpec is the specification of an Team
type TeamSpec struct {
	Label   string       `json:"label,omitempty" protobuf:"bytes,1,opt,name=label"`
	Kind    TeamKindType `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	Members []string     `json:"members,omitempty" protobuf:"bytes,3,opt,name=members"`
}

// TeamStatus is the status for an Team resource
type TeamStatus struct {
	ProvisionStatus TeamProvisionStatusType `json:"provisionStatus,omitempty"`
	Message         string                  `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TeamList is a list of TypeMeta resources
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Team `json:"items"`
}

// TeamKindType is the kind of an Team
type TeamKindType string

const (
	// TeamKindTypeCD specifies that the Team is a regular permanent one
	TeamKindTypeCD TeamKindType = "CD"
	// TeamKindTypeCI specifies that the Team is a regular permanent one
	TeamKindTypeCI TeamKindType = "CI"
)

// TeamProvisionStatusType is the kind of an Team
type TeamProvisionStatusType string

const (
	// TeamProvisionStatusNone provisioning not started yet
	TeamProvisionStatusNone TeamProvisionStatusType = ""

	// TeamProvisionStatusPending specifies that the Team is being provisioned
	TeamProvisionStatusPending TeamProvisionStatusType = "Pending"

	// TeamProvisionStatusComplete specifies that the Team has been provisioned
	TeamProvisionStatusComplete TeamProvisionStatusType = "Complete"

	// TeamProvisionStatusDeleting specifies that the Team is being deleted
	TeamProvisionStatusDeleting TeamProvisionStatusType = "Deleting"

	// TeamProvisionStatusError specifies that the Team provisioning failed with some error
	TeamProvisionStatusError TeamProvisionStatusType = "Error"
)
