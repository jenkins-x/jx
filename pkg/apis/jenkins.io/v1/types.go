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
	Version            string                 `json:"version,omitempty" protobuf:"bytes,3,opt,name=version"`
	Status             ActivityStatusType     `json:"status,omitempty" protobuf:"bytes,4,opt,name=status"`
	StartedTimestamp   *metav1.Time           `json:"startedTimestamp,omitempty" protobuf:"bytes,5,opt,name=startedTimestamp"`
	CompletedTimestamp *metav1.Time           `json:"completedTimestamp,omitempty" protobuf:"bytes,6,opt,name=completedTimestamp"`
	Steps              []PipelineActivityStep `json:"steps,omitempty" protobuf:"bytes,7,opt,name=steps"`
	BuildURL           string                 `json:"buildUrl,omitempty" protobuf:"bytes,8,opt,name=buildUrl"`
	BuildLogsURL       string                 `json:"buildLogsUrl,omitempty" protobuf:"bytes,9,opt,name=buildLogsUrl"`
	GitURL             string                 `json:"gitUrl,omitempty" protobuf:"bytes,10,opt,name=gitUrl"`
	GitRepository      string                 `json:"gitRepository,omitempty" protobuf:"bytes,10,opt,name=gitRepository"`
	GitOwner           string                 `json:"gitOwner,omitempty" protobuf:"bytes,10,opt,name=gitOwner"`
}

type PipelineActivityStep struct {
	Kind    ActivityStepKindType `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	Stage   *StageActivityStep   `json:"stage,omitempty" protobuf:"bytes,2,opt,name=stage"`
	Promote *PromoteActivityStep `json:"promote,omitempty" protobuf:"bytes,3,opt,name=promote"`
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

type PromoteActivityStep struct {
	CoreActivityStep

	Environment string                  `json:"environment,omitempty" protobuf:"bytes,1,opt,name=environment"`
	PullRequest *PromotePullRequestStep `json:"pullRequest,omitempty" protobuf:"bytes,2,opt,name=pullRequest"`
	Update      *PromoteUpdateStep      `json:"update,omitempty" protobuf:"bytes,3,opt,name=update"`
}

type GitStatus struct {
	URL    string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	Status string `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

type PromotePullRequestStep struct {
	CoreActivityStep

	PullRequestURL string `json:"pullRequestURL,omitempty" protobuf:"bytes,1,opt,name=pullRequestURL"`
	MergeCommitSHA string `json:"mergeCommitSHA,omitempty" protobuf:"bytes,2,opt,name=mergeCommitSHA"`
}

type PromoteUpdateStep struct {
	CoreActivityStep

	Statuses []GitStatus `json:"statuses,omitempty" protobuf:"bytes,1,opt,name=statuses"`
}

// PipelineActivityStatus is the status for an Environment resource
type PipelineActivityStatus struct {
	Version string `json:"version,omitempty"  protobuf:"bytes,1,opt,name=version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineActivityList is a list of PipelineActivity resources
type PipelineActivityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PipelineActivity `json:"items"`
}

// ActivityStepKindType
type ActivityStepKindType string

const (
	// ActivityStepKindTypeNone no kind yet
	ActivityStepKindTypeNone ActivityStepKindType = ""
	// ActivityStepKindTypeStage a group of low level steps
	ActivityStepKindTypeStage ActivityStepKindType = "Stage"
	// ActivityStepKindTypePromote a promote activity
	ActivityStepKindTypePromote ActivityStepKindType = "Promote"
)

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

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

type Release struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ReleaseSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status ReleaseStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReleaseList is a list of Release resources
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Release `json:"items"`
}

type ReleaseSpec struct {
	Name         string          `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Version      string          `json:"version,omitempty"  protobuf:"bytes,2,opt,name=version"`
	GitHttpURL   string          `json:"gitHttpUrl,omitempty"  protobuf:"bytes,3,opt,name=gitHttpUrl"`
	GitCloneURL  string          `json:"gitCloneUrl,omitempty"  protobuf:"bytes,4,opt,name=gitCloneUrl"`
	Commits      []CommitSummary `json:"commits,omitempty" protobuf:"bytes,5,opt,name=commits"`
	Issues       []IssueSummary  `json:"issues,omitempty" protobuf:"bytes,6,opt,name=issues"`
	PullRequests []IssueSummary  `json:"pullRequests,omitempty" protobuf:"bytes,7,opt,name=pullRequests"`
}

// ReleaseStatus is the status of a release
type ReleaseStatus struct {
	Status ReleaseStatusType `json:"status,omitempty"  protobuf:"bytes,1,opt,name=status"`
}

// IssueSummary is the summary of an issue
type IssueSummary struct {
	ID        string        `json:"id,omitempty"  protobuf:"bytes,1,opt,name=id"`
	URL       string        `json:"url,omitempty"  protobuf:"bytes,2,opt,name=url"`
	Title     string        `json:"title,omitempty"  protobuf:"bytes,3,opt,name=title"`
	Body      string        `json:"body,omitempty"  protobuf:"bytes,4,opt,name=body"`
	State     string        `json:"state,omitempty"  protobuf:"bytes,5,opt,name=state"`
	Message   string        `json:"message,omitempty"  protobuf:"bytes,6,opt,name=message"`
	User      *UserDetails  `json:"user,omitempty"  protobuf:"bytes,7,opt,name=user"`
	Assignees []UserDetails `json:"assignees,omitempty"  protobuf:"bytes,8,opt,name=assignees"`
	ClosedBy  *UserDetails  `json:"closedBy,omitempty"  protobuf:"bytes,9,opt,name=closedBy"`
}

// CommitSummary is the summary of a commit
type CommitSummary struct {
	Message   string       `json:"message,omitempty"  protobuf:"bytes,1,opt,name=message"`
	SHA       string       `json:"sha,omitempty"  protobuf:"bytes,2,opt,name=sha"`
	URL       string       `json:"url,omitempty"  protobuf:"bytes,3,opt,name=url"`
	Author    *UserDetails `json:"author,omitempty"  protobuf:"bytes,4,opt,name=author"`
	Committer *UserDetails `json:"committer,omitempty"  protobuf:"bytes,5,opt,name=committer"`
	Branch    string       `json:"branch,omitempty"  protobuf:"bytes,6,opt,name=branch"`
}

// UserDetails containers details of a user
type UserDetails struct {
	Login             string       `json:"login,omitempty"  protobuf:"bytes,1,opt,name=login"`
	Name              string       `json:"name,omitempty"  protobuf:"bytes,2,opt,name=name"`
	Email             string       `json:"email,omitempty"  protobuf:"bytes,3,opt,name=email"`
	CreationTimestamp *metav1.Time `json:"creationTimestamp,omitempty" protobuf:"bytes,4,opt,name=creationTimestamp"`
	URL               string       `json:"url,omitempty"  protobuf:"bytes,5,opt,name=url"`
	AvatarURL         string       `json:"avatarURL,omitempty"  protobuf:"bytes,6,opt,name=avatarURL"`
}

// ReleaseStatusType
type ReleaseStatusType string

const (
	// ReleaseStatusTypeNone an activity step has not started yet
	ReleaseStatusTypeNone ReleaseStatusType = ""
	// ReleaseStatusTypePending the release is pending
	ReleaseStatusTypePending ReleaseStatusType = "Pending"
	// ReleaseStatusTypeDeployed a release has been deployed
	ReleaseStatusTypeDeployed ReleaseStatusType = "Deployed"
	// ReleaseStatusTypeFailed release failed
	ReleaseStatusTypeFailed ReleaseStatusType = "Failed"
)
