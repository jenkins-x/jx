package v1

import (
	batchv1 "k8s.io/api/batch/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	WebHookEngine     WebHookEngineType     `json:"webHookEngine,omitempty" protobuf:"bytes,11,opt,name=webHookEngine"`
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

// PromotionEngineType is the type of promotion implementation the team uses
type PromotionEngineType string

const (
	PromotionEngineJenkins PromotionEngineType = "Jenkins"
	PromotionEngineProw    PromotionEngineType = "Prow"
)

// WebHookEngineType is the type of webhook processing implementation the team uses
type WebHookEngineType string

const (
	WebHookEngineNone    WebHookEngineType = ""
	WebHookEngineJenkins WebHookEngineType = "Jenkins"
	WebHookEngineProw    WebHookEngineType = "Prow"
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
	UseGitOps           bool                 `json:"useGitOps,omitempty" protobuf:"bytes,1,opt,name=useGitOps"`
	AskOnCreate         bool                 `json:"askOnCreate,omitempty" protobuf:"bytes,2,opt,name=askOnCreate"`
	BranchPatterns      string               `json:"branchPatterns,omitempty" protobuf:"bytes,3,opt,name=branchPatterns"`
	ForkBranchPatterns  string               `json:"forkBranchPatterns,omitempty" protobuf:"bytes,4,opt,name=forkBranchPatterns"`
	QuickstartLocations []QuickStartLocation `json:"quickstartLocations,omitempty" protobuf:"bytes,5,opt,name=quickstartLocations"`
	BuildPackURL        string               `json:"buildPackUrl,omitempty" protobuf:"bytes,6,opt,name=buildPackUrl"`
	BuildPackRef        string               `json:"buildPackRef,omitempty" protobuf:"bytes,7,opt,name=buildPackRef"`
	HelmBinary          string               `json:"helmBinary,omitempty" protobuf:"bytes,8,opt,name=helmBinary"`
	PostPreviewJobs     []batchv1.Job        `json:"postPreviewJobs,omitempty" protobuf:"bytes,9,opt,name=postPreviewJobs"`
	PromotionEngine     PromotionEngineType  `json:"promotionEngine,omitempty" protobuf:"bytes,10,opt,name=promotionEngine"`
	NoTiller            bool                 `json:"noTiller,omitempty" protobuf:"bytes,11,opt,name=noTiller"`
	HelmTemplate        bool                 `json:"helmTemplate,omitempty" protobuf:"bytes,12,opt,name=helmTemplate"`
	GitServer           string               `json:"gitServer,omitempty" protobuf:"bytes,13,opt,name=gitServer" command:"gitserver" commandUsage:"Default git server for new repositories"`
	Organisation        string               `json:"organisation,omitempty" protobuf:"bytes,14,opt,name=organisation" command:"organisation" commandUsage:"Default git organisation for new repositories"`
	EnvOrganisation     string               `json:"envOrganisation,omitempty" protobuf:"bytes,14,opt,name=envOrganisation" command:"envOrganisation" commandUsage:"Default git organisation for new environment repositories"`
	PipelineUsername    string               `json:"pipelineUsername,omitempty" protobuf:"bytes,15,opt,name=pipelineUsername" command:"pipelineusername" commandUsage:"User used by pipeline. Is given write permission on new repositories."`
	DockerRegistryOrg   string               `json:"dockerRegistryOrg,omitempty" protobuf:"bytes,16,opt,name=dockerRegistryOrg" command:"dockerregistryorg" commandUsage:"Docker registry organisation used for new projects in Jenkins X."`
	GitPrivate          bool                 `json:"gitPrivate,omitempty" protobuf:"bytes,17,opt,name=gitPrivate" command:"gitprivate" commandUsage:"Are new repositories private by default"`
	KubeProvider        string               `json:"kubeProvider,omitempty" protobuf:"bytes,18,opt,name=kubeProvider"`
	AppsRepository      string               `json:"appsRepository,omitempty" protobuf:"bytes,19,opt,name=appsRepository"`
	BuildPackName       string               `json:"buildPackName,omitempty" protobuf:"bytes,20,opt,name=buildPackName"`
	StorageLocations    []StorageLocation    `json:"storageLocations,omitempty" protobuf:"bytes,21,opt,name=storageLocations"`
}

// StorageLocation
type StorageLocation struct {
	Classifier string `json:"classifier,omitempty" protobuf:"bytes,1,opt,name=classifier"`
	GitURL     string `json:"gitUrl,omitempty" protobuf:"bytes,2,opt,name=gitUrl"`
	GitBranch  string `json:"gitBranch,omitempty" protobuf:"bytes,3,opt,name=gitBranch"`
	BucketURL  string `json:"bucketUrl,omitempty" protobuf:"bytes,4,opt,name=bucketUrl"`
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

// EnvironmentRoleBinding is like a vanilla RoleBinding but applies to a set of Namespaces based on an Environment filter
// so that roles can be bound to multiple namespaces easily.
//
// For example to specify the binding of roles on all Preview environments or on all permanent environments.
type EnvironmentRoleBinding struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EnvironmentRoleBindingSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EnvironmentRoleBindingStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EnvironmentRoleBindingSpec is the specification of an EnvironmentRoleBinding
type EnvironmentRoleBindingSpec struct {
	// Subjects holds references to the objects the role applies to.
	Subjects []rbacv1.Subject `json:"subjects" protobuf:"bytes,2,rep,name=subjects"`

	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef rbacv1.RoleRef `json:"roleRef" protobuf:"bytes,3,opt,name=roleRef"`

	// specifies which sets of environments this binding applies to
	Environments []EnvironmentFilter `json:"environments,omitempty" protobuf:"bytes,4,opt,name=environments"`
}

// EnvironmentFilter specifies the environments to apply the role binding to
type EnvironmentFilter struct {
	Kind     EnvironmentKindType `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	Includes []string            `json:"includes,omitempty" protobuf:"bytes,2,opt,name=includes"`
	Excludes []string            `json:"excludes,omitempty" protobuf:"bytes,3,opt,name=excludes"`
}

// EnvironmentRoleBindingStatus is the status for an EnvironmentRoleBinding resource
type EnvironmentRoleBindingStatus struct {
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvironmentRoleBindingList is a list of TypeMeta resources
type EnvironmentRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EnvironmentRoleBinding `json:"items"`
}

// StorageLocationOrDefault returns the storage location if there is one or returns the default storage configuration
func (t *TeamSettings) StorageLocationOrDefault(classifier string) *StorageLocation {
	for idx, sl := range t.StorageLocations {
		if sl.Classifier == classifier {
			return &t.StorageLocations[idx]
		}
	}
	return t.StorageLocation("default")
}

// StorageLocation returns the storage location, lazily creating one if one does not already exist
func (t *TeamSettings) StorageLocation(classifier string) *StorageLocation {
	for idx, sl := range t.StorageLocations {
		if sl.Classifier == classifier {
			return &t.StorageLocations[idx]
		}
	}
	t.StorageLocations = append(t.StorageLocations, StorageLocation{
		Classifier: classifier,
	})
	return &t.StorageLocations[len(t.StorageLocations)-1]
}

// SetStorageLocation stores the given storage location in the team settings
func (t *TeamSettings) SetStorageLocation(classifier string, storage StorageLocation) {
	storage.Classifier = classifier
	for idx, sl := range t.StorageLocations {
		if sl.Classifier == classifier {
			t.StorageLocations[idx] = storage
			return
		}
	}
	t.StorageLocations = append(t.StorageLocations, storage)
}

// IsEmpty returns true if the storage location is empty
func (s *StorageLocation) IsEmpty() bool {
	return s.GitURL == "" && s.BucketURL == ""
}

// Description returns the textual description of the storage location
func (s *StorageLocation) Description() string {
	if s.GitURL != "" {
		return s.GitURL + " branch: " + s.GetGitBranch()
	}
	if s.BucketURL != "" {
		return s.BucketURL
	}
	return "current git repo"
}

// GetGitBranch returns the git branch to use when using git storage
func (s *StorageLocation) GetGitBranch() string {
	branch := s.GitBranch
	if branch == "" {
		branch = "gh-pages"
	}
	return branch
}
