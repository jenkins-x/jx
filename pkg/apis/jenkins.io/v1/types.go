package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

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
	Kind              EnvironmentKindType   `json:"kind,omitempty" protobuf:"bytes,7,opt,name=kind"`
	PullRequestURL    string                `json:"pullRequestURL,omitempty" protobuf:"bytes,8,opt,name=pullRequestURL"`
	TeamSettings      TeamSettings          `json:"teamSettings,omitempty" protobuf:"bytes,9,opt,name=teamSettings"`
	//PreviewGitSpec    PreviewGitSpec        `json:"previewGitSpec,omitempty" protobuf:"bytes,8,opt,name=previewGitSpec"`
}

// EnvironmentStatus is the status for an Environment resource
type EnvironmentStatus struct {
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvironmentList is a list of TypeMeta resources
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
	// EnvironmentKindTypeTest specifies that an environment is a temporary one for a test
	EnvironmentKindTypeTest EnvironmentKindType = "Test"
	// EnvironmentKindTypeEdit specifies that an environment is a developers editing workspace
	EnvironmentKindTypeEdit EnvironmentKindType = "Edit"
)

// IsPermanent returns true if this environment is permanent
func (e EnvironmentKindType) IsPermanent() bool {
	switch e {
	case EnvironmentKindTypePreview, EnvironmentKindTypeTest, EnvironmentKindTypeEdit:
		return false
	default:
		return true
	}
}

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

type TeamSettings struct {
	UseGitOPs   bool `json:"useGitOps,omitempty" protobuf:"bytes,1,opt,name=useGitOps"`
	AskOnCreate bool `json:"askOnCreate,omitempty" protobuf:"bytes,1,opt,name=askOnCreate"`
}
type PreviewGitSpec struct {
	Name string   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	URL  string   `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
	User UserSpec `json:"user,omitempty" protobuf:"bytes,3,opt,name=user"`
}

type UserSpec struct {
	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	Name     string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	LinkURL  string `json:"linkUrl,omitempty" protobuf:"bytes,3,opt,name=linkUrl"`
	ImageURL string `json:"imageUrl,omitempty" protobuf:"bytes,4,opt,name=imageUrl"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

type PipelineActivity struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PipelineActivitySpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PipelineActivityStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type PipelineActivitySpec struct {
	Pipeline           string                 `json:"pipeline,omitempty" protobuf:"bytes,1,opt,name=pipeline"`
	Build              string                 `json:"build,omitempty" protobuf:"bytes,2,opt,name=build"`
	Status             ActivityStatusType     `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
	StartedTimestamp   *metav1.Time           `json:"startedTimestamp,omitempty" protobuf:"bytes,4,opt,name=startedTimestamp"`
	CompletedTimestamp *metav1.Time           `json:"completedTimestamp,omitempty" protobuf:"bytes,5,opt,name=completedTimestamp"`
	Steps              []PipelineActivityStep `json:"steps,omitempty" protobuf:"bytes,6,opt,name=steps"`
}

type CoreActivityStep struct {
	Name               string             `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Description        string             `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`
	Status             ActivityStatusType `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
	StartedTimestamp   *metav1.Time       `json:"startedTimestamp,omitempty" protobuf:"bytes,4,opt,name=startedTimestamp"`
	CompletedTimestamp *metav1.Time       `json:"completedTimestamp,omitempty" protobuf:"bytes,5,opt,name=completedTimestamp"`
}

type StageActivityStep struct {
	CoreActivityStep

	Steps []CoreActivityStep `json:"steps,omitempty" protobuf:"bytes,1,opt,name=steps"`
}

type PullRequestStatus struct {
	URL    string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	Status string `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

type PromotePullRequestStep struct {
	CoreActivityStep

	Environment    string              `json:"environment,omitempty" protobuf:"bytes,1,opt,name=environment"`
	PullRequestURL string              `json:"pullRequestURL,omitempty" protobuf:"bytes,2,opt,name=pullRequestURL"`
	Statuses       []PullRequestStatus `json:"statuses,omitempty" protobuf:"bytes,3,opt,name=statuses"`
	MergeCommitSHA string              `json:"mergeCommitSHA,omitempty" protobuf:"bytes,4,opt,name=mergeCommitSHA"`
}

type PipelineActivityStep struct {
	Stage              *CoreActivityStep       `json:"stage,omitempty" protobuf:"bytes,1,opt,name=stage"`
	Step               *CoreActivityStep       `json:"step,omitempty" protobuf:"bytes,1,opt,name=step"`
	PromotePullRequest *PromotePullRequestStep `json:"promotePullRequest,omitempty" protobuf:"bytes,3,opt,name=promotePullRequest"`
	Promote            *PromotePullRequestStep `json:"promote,omitempty" protobuf:"bytes,4,opt,name=promote"`
}

// PipelineActivityStatus is the status for an Environment resource
type PipelineActivityStatus struct {
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineActivityList is a list of PipelineActivity resources
type PipelineActivityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PipelineActivity `json:"items"`
}

// ActivityStatusType
type ActivityStatusType string

const (
	// ActivityStatusTypeNone an activity step has not started yet
	ActivityStatusTypeNone ActivityStatusType = ""
	// ActivityStatusTypePending an activity step is waiting to start
	ActivityStatusTypePending ActivityStatusType = "Pending"
	// ActivityStatusTypeRunning an activity is running
	ActivityStatusTypeRunning ActivityStatusType = "Running"
	// ActivityStatusTypeSucceeded an activity completed successfully
	ActivityStatusTypeSucceeded ActivityStatusType = "Succeeded"
	// ActivityStatusTypeFailed an activity failed
	ActivityStatusTypeFailed ActivityStatusType = "Failed"
	// ActivityStatusTypeWaitingForApproval an activity is waiting for approval
	ActivityStatusTypeWaitingForApproval ActivityStatusType = "WaitingForApproval"
	// ActivityStatusTypeError there is some error with an activity
	ActivityStatusTypeError ActivityStatusType = "Error"
)

func (s ActivityStatusType) String() string {
	return string(s)
}
