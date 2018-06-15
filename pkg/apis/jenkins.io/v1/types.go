package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Environment represents an environment like Dev, Test, Staging, Production where code lives
type Environment struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EnvironmentSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EnvironmentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EnvironmentSpec is the specification of an Environment
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
	PreviewGitSpec    PreviewGitSpec        `json:"previewGitInfo,omitempty" protobuf:"bytes,10,opt,name=previewGitInfo"`
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

// PromotionStrategyType is the type of a promotion strategy
type PromotionStrategyType string

const (
	// PromotionStrategyTypeManual specifies that promotion happens manually
	PromotionStrategyTypeManual PromotionStrategyType = "Manual"
	// PromotionStrategyTypeAutomatic specifies that promotion happens automatically
	PromotionStrategyTypeAutomatic PromotionStrategyType = "Auto"
	// PromotionStrategyTypeNever specifies that promotion is disabled for this environment
	PromotionStrategyTypeNever PromotionStrategyType = "Never"
)

// EnvironmentKindType is the kind of an environment
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
	// EnvironmentKindTypeDevelopment specifies that an environment is a development environment; for developer tools like Jenkins, Nexus etc
	EnvironmentKindTypeDevelopment EnvironmentKindType = "Development"
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

// EnvironmentRepositoryType is the repository type
type EnvironmentRepositoryType string

const (
	// EnvironmentRepositoryTypeGit specifies that a git repository is used
	EnvironmentRepositoryTypeGit EnvironmentRepositoryType = "Git"
)

// EnvironmentRepository is the repository for an environment using GitOps
type EnvironmentRepository struct {
	Kind EnvironmentRepositoryType `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	URL  string                    `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
	Ref  string                    `json:"ref,omitempty" protobuf:"bytes,3,opt,name=ref"`
}

// TeamSettings the default settings for a team
type TeamSettings struct {
	UseGitOPs           bool                 `json:"useGitOps,omitempty" protobuf:"bytes,1,opt,name=useGitOps"`
	AskOnCreate         bool                 `json:"askOnCreate,omitempty" protobuf:"bytes,2,opt,name=askOnCreate"`
	BranchPatterns      string               `json:"branchPatterns,omitempty" protobuf:"bytes,3,opt,name=branchPatterns"`
	ForkBranchPatterns  string               `json:"forkBranchPatterns,omitempty" protobuf:"bytes,4,opt,name=forkBranchPatterns"`
	QuickstartLocations []QuickStartLocation `json:"quickstartLocations,omitempty" protobuf:"bytes,5,opt,name=quickstartLocations"`
	BuildPackURL        string               `json:"buildPackUrl,omitempty" protobuf:"bytes,6,opt,name=buildPackUrl"`
	BuildPackRef        string               `json:"buildPackRef,omitempty" protobuf:"bytes,7,opt,name=buildPackRef"`
	HelmBinary          string               `json:"helmBinary,omitempty" protobuf:"bytes,8,opt,name=helmBinary"`
}

// QuickStartLocation
type QuickStartLocation struct {
	GitURL   string   `json:"gitUrl,omitempty" protobuf:"bytes,1,opt,name=gitUrl"`
	GitKind  string   `json:"gitKind,omitempty" protobuf:"bytes,2,opt,name=gitKind"`
	Owner    string   `json:"owner,omitempty" protobuf:"bytes,3,opt,name=owner"`
	Includes []string `json:"includes,omitempty" protobuf:"bytes,4,opt,name=includes"`
	Excludes []string `json:"excludes,omitempty" protobuf:"bytes,5,opt,name=excludes"`
}

// PreviewGitSpec is the preview git branch/pull request details
type PreviewGitSpec struct {
	Name            string   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	URL             string   `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
	User            UserSpec `json:"user,omitempty" protobuf:"bytes,3,opt,name=user"`
	Title           string   `json:"title,omitempty" protobuf:"bytes,4,opt,name=title"`
	Description     string   `json:"description,omitempty" protobuf:"bytes,5,opt,name=description"`
	BuildStatus     string   `json:"buildStatus,omitempty" protobuf:"bytes,6,opt,name=buildStatus"`
	BuildStatusURL  string   `json:"buildStatusUrl,omitempty" protobuf:"bytes,7,opt,name=buildStatusUrl"`
	ApplicationName string   `json:"appName,omitempty" protobuf:"bytes,8,opt,name=appName"`
	ApplicationURL  string   `json:"applicationURL,omitempty" protobuf:"bytes,9,opt,name=applicationURL"`
}

// UserSpec is the user details
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

// PipelineActivity represents pipeline activity for a particular run of a pipeline
type PipelineActivity struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PipelineActivitySpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PipelineActivityStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// PipelineActivitySpec is the specification of the pipeline activity
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
	ReleaseNotesURL    string                 `json:"releaseNotesURL,omitempty" protobuf:"bytes,11,opt,name=releaseNotesURL"`
}

// PipelineActivityStep represents a step in a pipeline activity
type PipelineActivityStep struct {
	Kind    ActivityStepKindType `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	Stage   *StageActivityStep   `json:"stage,omitempty" protobuf:"bytes,2,opt,name=stage"`
	Promote *PromoteActivityStep `json:"promote,omitempty" protobuf:"bytes,3,opt,name=promote"`
	Preview *PreviewActivityStep `json:"preview,omitempty" protobuf:"bytes,4,opt,name=preview"`
}

// CoreActivityStep is a base step included in Stages of a pipeline or other kinds of step
type CoreActivityStep struct {
	Name               string             `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Description        string             `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`
	Status             ActivityStatusType `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
	StartedTimestamp   *metav1.Time       `json:"startedTimestamp,omitempty" protobuf:"bytes,4,opt,name=startedTimestamp"`
	CompletedTimestamp *metav1.Time       `json:"completedTimestamp,omitempty" protobuf:"bytes,5,opt,name=completedTimestamp"`
}

// StageActivityStep represents a stage of zero to more sub steps in a jenkins pipeline
type StageActivityStep struct {
	CoreActivityStep

	Steps []CoreActivityStep `json:"steps,omitempty" protobuf:"bytes,1,opt,name=steps"`
}

// PreviewActivityStep is the step of creating a preview environment as part of a Pull Request pipeine
type PreviewActivityStep struct {
	CoreActivityStep

	Environment    string `json:"environment,omitempty" protobuf:"bytes,1,opt,name=environment"`
	PullRequestURL string `json:"pullRequestURL,omitempty" protobuf:"bytes,2,opt,name=pullRequestURL"`
	ApplicationURL string `json:"applicationURL,omitempty" protobuf:"bytes,3,opt,name=applicationURL"`
}

// PromoteActivityStep is the step of promoting a version of an application to an environment
type PromoteActivityStep struct {
	CoreActivityStep

	Environment    string                  `json:"environment,omitempty" protobuf:"bytes,1,opt,name=environment"`
	PullRequest    *PromotePullRequestStep `json:"pullRequest,omitempty" protobuf:"bytes,2,opt,name=pullRequest"`
	Update         *PromoteUpdateStep      `json:"update,omitempty" protobuf:"bytes,3,opt,name=update"`
	ApplicationURL string                  `json:"applicationURL,omitempty" protobuf:"bytes,4,opt,name=environment"`
}

// GitStatus the status of a git commit in terms of CI/CD
type GitStatus struct {
	URL    string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	Status string `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

// PromotePullRequestStep is the step for promoting a version to an environment by raising a Pull Request on the
// git repository of the environment
type PromotePullRequestStep struct {
	CoreActivityStep

	PullRequestURL string `json:"pullRequestURL,omitempty" protobuf:"bytes,1,opt,name=pullRequestURL"`
	MergeCommitSHA string `json:"mergeCommitSHA,omitempty" protobuf:"bytes,2,opt,name=mergeCommitSHA"`
}

// PromoteUpdateStep is the step for updating a promotion after the Pull Request merges to master
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

// ActivityStepKindType is a kind of step
type ActivityStepKindType string

const (
	// ActivityStepKindTypeNone no kind yet
	ActivityStepKindTypeNone ActivityStepKindType = ""
	// ActivityStepKindTypeStage a group of low level steps
	ActivityStepKindTypeStage ActivityStepKindType = "Stage"
	// ActivityStepKindTypePreview a promote activity
	ActivityStepKindTypePreview ActivityStepKindType = "Preview"
	// ActivityStepKindTypePromote a promote activity
	ActivityStepKindTypePromote ActivityStepKindType = "Promote"
)

// ActivityStatusType is the status of an activity; usually succeeded or failed/error on completion
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

// Release represents a single version of an app that has been released
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

// ReleaseSpec is the specification of the Release
type ReleaseSpec struct {
	Name            string          `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Version         string          `json:"version,omitempty"  protobuf:"bytes,2,opt,name=version"`
	GitHTTPURL      string          `json:"gitHttpUrl,omitempty"  protobuf:"bytes,3,opt,name=gitHttpUrl"`
	GitCloneURL     string          `json:"gitCloneUrl,omitempty"  protobuf:"bytes,4,opt,name=gitCloneUrl"`
	Commits         []CommitSummary `json:"commits,omitempty" protobuf:"bytes,5,opt,name=commits"`
	Issues          []IssueSummary  `json:"issues,omitempty" protobuf:"bytes,6,opt,name=issues"`
	PullRequests    []IssueSummary  `json:"pullRequests,omitempty" protobuf:"bytes,7,opt,name=pullRequests"`
	ReleaseNotesURL string          `json:"releaseNotesURL,omitempty" protobuf:"bytes,8,opt,name=releaseNotesURL"`
	GitRepository   string          `json:"gitRepository,omitempty" protobuf:"bytes,9,opt,name=gitRepository"`
	GitOwner        string          `json:"gitOwner,omitempty" protobuf:"bytes,10,opt,name=gitOwner"`
}

// ReleaseStatus is the status of a release
type ReleaseStatus struct {
	Status ReleaseStatusType `json:"status,omitempty"  protobuf:"bytes,1,opt,name=status"`
}

// IssueSummary is the summary of an issue
type IssueSummary struct {
	ID                string        `json:"id,omitempty"  protobuf:"bytes,1,opt,name=id"`
	URL               string        `json:"url,omitempty"  protobuf:"bytes,2,opt,name=url"`
	Title             string        `json:"title,omitempty"  protobuf:"bytes,3,opt,name=title"`
	Body              string        `json:"body,omitempty"  protobuf:"bytes,4,opt,name=body"`
	State             string        `json:"state,omitempty"  protobuf:"bytes,5,opt,name=state"`
	Message           string        `json:"message,omitempty"  protobuf:"bytes,6,opt,name=message"`
	User              *UserDetails  `json:"user,omitempty"  protobuf:"bytes,7,opt,name=user"`
	Assignees         []UserDetails `json:"assignees,omitempty"  protobuf:"bytes,8,opt,name=assignees"`
	ClosedBy          *UserDetails  `json:"closedBy,omitempty"  protobuf:"bytes,9,opt,name=closedBy"`
	CreationTimestamp *metav1.Time  `json:"creationTimestamp,omitempty" protobuf:"bytes,10,opt,name=creationTimestamp"`
	Labels            []IssueLabel  `json:"labels,omitempty" protobuf:"bytes,11,opt,name=labels"`
}

type IssueLabel struct {
	URL   string `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	Name  string `json:"url,omitempty"  protobuf:"bytes,2,opt,name=url"`
	Color string `json:"color,omitempty"  protobuf:"bytes,3,opt,name=color"`
}

// CommitSummary is the summary of a commit
type CommitSummary struct {
	Message   string       `json:"message,omitempty"  protobuf:"bytes,1,opt,name=message"`
	SHA       string       `json:"sha,omitempty"  protobuf:"bytes,2,opt,name=sha"`
	URL       string       `json:"url,omitempty"  protobuf:"bytes,3,opt,name=url"`
	Author    *UserDetails `json:"author,omitempty"  protobuf:"bytes,4,opt,name=author"`
	Committer *UserDetails `json:"committer,omitempty"  protobuf:"bytes,5,opt,name=committer"`
	Branch    string       `json:"branch,omitempty"  protobuf:"bytes,6,opt,name=branch"`
	IssueIDs  []string     `json:"issueIds,omitempty"  protobuf:"bytes,7,opt,name=issueIds"`
}

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

// ReleaseStatusType is the status of a release; usually deployed or failed at completion
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

// IsClosed returns true if this issue is closed or fixed
func (i *IssueSummary) IsClosed() bool {
	lower := strings.ToLower(i.State)
	return strings.HasPrefix(lower, "clos") || strings.HasPrefix(lower, "fix")
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// GitService represents a git provider so we can map the host name to a kinda of git service
type GitService struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec GitServiceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// GitServiceSpec is the specification of an GitService
type GitServiceSpec struct {
	GitKind string `json:"gitKind,omitempty" protobuf:"bytes,1,opt,name=gitKind"`
	URL     string `json:"url,omitempty" protobuf:"bytes,2,opt,name=host"`
	Name    string `json:"name,omitempty" protobuf:"bytes,3,opt,name=host"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitServiceList is a list of GitService resources
type GitServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []GitService `json:"items"`
}
