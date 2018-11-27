package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// BuildPack represents a set of language specific build packs and associated quickstarts
type BuildPack struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec BuildPackSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// BuildPackSpec is the specification of an BuildPack
type BuildPackSpec struct {
	Label               string               `json:"label,omitempty" protobuf:"bytes,1,opt,name=label"`
	GitURL              string               `json:"gitUrl,omitempty" protobuf:"bytes,2,opt,name=gitUrl"`
	GitRef              string               `json:"gitRef,omitempty" protobuf:"bytes,3,opt,name=gitRef"`
	QuickstartLocations []QuickStartLocation `json:"quickstartLocations,omitempty" protobuf:"bytes,4,opt,name=quickstartLocations"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildPackList is a list of TypeMeta resources
type BuildPackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BuildPack `json:"items"`
}
