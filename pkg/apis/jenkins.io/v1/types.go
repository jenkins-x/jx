package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Environment struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EnvironmentSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EnvironmentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type EnvironmentSpec struct {
	Label             string                `json:"label,omitempty" protobuf:"bytes,1,opt,name=label"`
	Namespace         string                `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	Cluster           string                `json:"cluster,omitempty" protobuf:"bytes,3,opt,name=cluster"`
	PromotionStrategy PromotionStrategyType `json:"promotionStrategy,omitempty" protobuf:"bytes,4,opt,name=promotionStrategy"`
	Source            EnvironmentRepository `json:"source,omitempty" protobuf:"bytes,5,opt,name=source"`
	Order             int32                 `json:"order,omitempty" protobuf:"bytes,6,opt,name=order"`
	PreviewGitSpec    PreviewGitSpec        `json:"previewGitSpec,omitempty" protobuf:"bytes,7,opt,name=previewGitSpec"`
	Kind              EnvironmentKindType   `json:"promotionStrategy,omitempty" protobuf:"bytes,4,opt,name=promotionStrategy"`
}

// EnvironmentStatus is the status for an Envirnment resource
type EnvironmentStatus struct {
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvironmentList is a list of Example resources
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Environment `json:"items"`
}

// Promotion Strategy Type string
type PromotionStrategyType string

const (
	// PromotionStrategyTypeManual specifies that promotion happens manually
	PromotionStrategyTypeManual PromotionStrategyType = "Manual"
	// PromotionStrategyTypeAutomatic specifies that promotion happens automatically
	PromotionStrategyTypeAutomatic PromotionStrategyType = "Auto"
	// PromotionStrategyTypeNever specifies that promotion is disabled for this environment
	PromotionStrategyTypeNever PromotionStrategyType = "Never"
)

// Environment Kind  Type string
type EnvironmentKindType string

const (
	// EnvironmentKindTypePermanent specifies that the environment is a regular permanent one
	EnvironmentKindTypePermanent EnvironmentKindType = "Permanent"
	// EnvironmentKindTypePreview specifies that an environment is a Preview environment that lasts as long as a Pull Request
	EnvironmentKindTypePreview EnvironmentKindType = "Preview"
	// EnvironmentKindTypeNever specifies that promotion is disabled for this environment
	EnvironmentKindTypeNever EnvironmentKindType = "Never"
)

// PromotionStrategyTypeValues is the list of all values
var PromotionStrategyTypeValues = []string{
	string(PromotionStrategyTypeAutomatic),
	string(PromotionStrategyTypeManual),
	string(PromotionStrategyTypeNever),
}

// Environment Repository Type string
type EnvironmentRepositoryType string

const (
	// EnvironmentRepositoryTypeGit specifies that a git repository is used
	EnvironmentRepositoryTypeGit EnvironmentRepositoryType = "Git"
)

type EnvironmentRepository struct {
	Kind EnvironmentRepositoryType `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	URL  string                    `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
	Ref  string                    `json:"ref,omitempty" protobuf:"bytes,3,opt,name=ref"`
}

type PreviewGitSpec struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	URL  string `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
}
