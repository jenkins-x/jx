package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// User represents a git user so we have a cache to find by email address
type User struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	User UserDetails `json:"user,omitempty" protobuf:"bytes,2,opt,name=user"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList is a list of User resources
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}

// UserDetails containers details of a user
type UserDetails struct {
	Login             string       `json:"login,omitempty"  protobuf:"bytes,1,opt,name=login"`
	Name              string       `json:"name,omitempty"  protobuf:"bytes,2,opt,name=name"`
	Email             string       `json:"email,omitempty"  protobuf:"bytes,3,opt,name=email"`
	CreationTimestamp *metav1.Time `json:"creationTimestamp,omitempty" protobuf:"bytes,4,opt,name=creationTimestamp"`
	URL               string       `json:"url,omitempty"  protobuf:"bytes,5,opt,name=url"`
	AvatarURL         string       `json:"avatarUrl,omitempty"  protobuf:"bytes,6,opt,name=avatarUrl"`
}
