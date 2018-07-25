package codeship

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// ProjectType represents Codeship project types (Basic and Pro)
type ProjectType int

const (
	// ProjectTypeBasic represents a Codeship Basic project type
	ProjectTypeBasic ProjectType = iota
	// ProjectTypePro represents a Codeship Pro project type
	ProjectTypePro
)

var (
	_projectTypeValueToName = map[ProjectType]string{
		ProjectTypeBasic: "basic",
		ProjectTypePro:   "pro",
	}
	_projectTypeNameToValue = map[string]ProjectType{
		"basic": ProjectTypeBasic,
		"pro":   ProjectTypePro,
	}
)

func (t ProjectType) String() string {
	return _projectTypeValueToName[t]
}

// MarshalJSON marshals a ProjectType to JSON
func (t ProjectType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON unmarshals JSON to a ProjectType
func (t *ProjectType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("ProjectType should be a string, got %T", data)
	}
	v, ok := _projectTypeNameToValue[s]
	if !ok {
		return fmt.Errorf("invalid ProjectType: %s", s)
	}
	*t = v
	return nil
}

// DeploymentBranch structure for DeploymentBranch object for a Basic Project
type DeploymentBranch struct {
	BranchName string `json:"branch_name,omitempty"`
	MatchMode  string `json:"match_mode,omitempty"`
}

// DeploymentPipeline structure for DeploymentPipeline object for a Basic Project
type DeploymentPipeline struct {
	Branch   DeploymentBranch       `json:"branch,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Position int                    `json:"position,omitempty"`
}

// EnvironmentVariable structure for EnvironmentVariable object for a Basic Project
type EnvironmentVariable struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// NotificationOptions structure for NotificationOptions object for a Project
type NotificationOptions struct {
	Key  string `json:"key,omitempty"`
	URL  string `json:"url,omitempty"`
	Room string `json:"room,omitempty"`
}

// NotificationRule structure for NotificationRule object for a Project
type NotificationRule struct {
	Branch        string              `json:"branch,omitempty"`
	BranchMatch   string              `json:"branch_match,omitempty"`
	Notifier      string              `json:"notifier,omitempty"`
	Options       NotificationOptions `json:"options,omitempty"`
	BuildStatuses []string            `json:"build_statuses,omitempty"`
	Target        string              `json:"target,omitempty"`
}

// TestPipeline structure for Project object
type TestPipeline struct {
	Commands []string `json:"commands,omitempty"`
	Name     string   `json:"name,omitempty"`
}

// Project structure for Project object
type Project struct {
	AesKey               string                `json:"aes_key,omitempty"`
	AuthenticationUser   string                `json:"authentication_user,omitempty"`
	CreatedAt            time.Time             `json:"created_at,omitempty"`
	DeploymentPipelines  []DeploymentPipeline  `json:"deployment_pipelines,omitempty"`
	EnvironmentVariables []EnvironmentVariable `json:"environment_variables,omitempty"`
	ID                   uint                  `json:"id,omitempty"`
	Name                 string                `json:"name,omitempty"`
	NotificationRules    []NotificationRule    `json:"notification_rules,omitempty"`
	OrganizationUUID     string                `json:"organization_uuid,omitempty"`
	RepositoryProvider   string                `json:"repository_provider,omitempty"`
	RepositoryURL        string                `json:"repository_url,omitempty"`
	SetupCommands        []string              `json:"setup_commands,omitempty"`
	SSHKey               string                `json:"ssh_key,omitempty"`
	TeamIDs              []int                 `json:"team_ids,omitempty"`
	TestPipelines        []TestPipeline        `json:"test_pipelines,omitempty"`
	Type                 ProjectType           `json:"type"`
	UpdatedAt            time.Time             `json:"updated_at,omitempty"`
	UUID                 string                `json:"uuid,omitempty"`
}

// ProjectCreateRequest structure for creating a Project
type ProjectCreateRequest struct {
	DeploymentPipelines  []DeploymentPipeline  `json:"deployment_pipelines,omitempty"`
	EnvironmentVariables []EnvironmentVariable `json:"environment_variables,omitempty"`
	NotificationRules    []NotificationRule    `json:"notification_rules,omitempty"`
	RepositoryURL        string                `json:"repository_url,omitempty"`
	SetupCommands        []string              `json:"setup_commands,omitempty"`
	TeamIDs              []int                 `json:"team_ids,omitempty"`
	TestPipelines        []TestPipeline        `json:"test_pipelines,omitempty"`
	Type                 ProjectType           `json:"type"`
}

// ProjectUpdateRequest structure for updating a Project
type ProjectUpdateRequest struct {
	EnvironmentVariables []EnvironmentVariable `json:"environment_variables,omitempty"`
	NotificationRules    []NotificationRule    `json:"notification_rules,omitempty"`
	SetupCommands        []string              `json:"setup_commands,omitempty"`
	TeamIDs              []int                 `json:"team_ids,omitempty"`
	Type                 ProjectType           `json:"type"`
}

// ProjectList holds a list of Project objects
type ProjectList struct {
	Projects []Project `json:"projects"`
	pagination
}

type projectResponse struct {
	Project Project `json:"project"`
}

// ListProjects fetches a list of projects
//
// Codeship API docs: https://apidocs.codeship.com/v2/projects/list-projects
func (o *Organization) ListProjects(ctx context.Context, opts ...PaginationOption) (ProjectList, Response, error) {
	path, err := paginate(fmt.Sprintf("/organizations/%s/projects", o.UUID), opts...)
	if err != nil {
		return ProjectList{}, Response{}, errors.Wrap(err, "unable to list projects")
	}

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return ProjectList{}, resp, errors.Wrap(err, "unable to list projects")
	}

	var projects ProjectList
	if err = json.Unmarshal(body, &projects); err != nil {
		return ProjectList{}, resp, errors.Wrap(err, "unable to unmarshal response into ProjectList")
	}

	return projects, resp, nil
}

// GetProject fetches a project by UUID
//
// Codeship API docs: https://apidocs.codeship.com/v2/projects/get-project
func (o *Organization) GetProject(ctx context.Context, projectUUID string) (Project, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s", o.UUID, projectUUID)

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to get project")
	}

	var project projectResponse
	if err = json.Unmarshal(body, &project); err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to unmarshal response into Project")
	}

	return project.Project, resp, nil
}

// CreateProject creates a new project
//
// Codeship API docs: https://apidocs.codeship.com/v2/projects/create-project
func (o *Organization) CreateProject(ctx context.Context, p ProjectCreateRequest) (Project, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects", o.UUID)

	body, resp, err := o.client.request(ctx, "POST", path, p)
	if err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to create project")
	}

	var project projectResponse
	if err = json.Unmarshal(body, &project); err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to unmarshal response into Project")
	}

	return project.Project, resp, nil
}

// UpdateProject updates an existing project
//
// Codeship API docs: https://apidocs.codeship.com/v2/projects/update-project
func (o *Organization) UpdateProject(ctx context.Context, projectUUID string, p ProjectUpdateRequest) (Project, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s", o.UUID, projectUUID)

	body, resp, err := o.client.request(ctx, "PUT", path, p)
	if err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to update project")
	}

	var project projectResponse
	if err = json.Unmarshal(body, &project); err != nil {
		return Project{}, resp, errors.Wrap(err, "unable to unmarshal response into Project")
	}

	return project.Project, resp, nil
}
