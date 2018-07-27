package codeship

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// BuildLinks structure of BuildLinks object for a Build
type BuildLinks struct {
	Pipelines string `json:"pipelines,omitempty"`
	Services  string `json:"services,omitempty"`
	Steps     string `json:"steps,omitempty"`
}

// Build structure of Build object
type Build struct {
	AllocatedAt      time.Time  `json:"allocated_at,omitempty"`
	Branch           string     `json:"branch,omitempty"`
	CommitMessage    string     `json:"commit_message,omitempty"`
	CommitSha        string     `json:"commit_sha,omitempty"`
	FinishedAt       time.Time  `json:"finished_at,omitempty"`
	Links            BuildLinks `json:"links,omitempty"`
	OrganizationUUID string     `json:"organization_uuid,omitempty"`
	ProjectID        uint       `json:"project_id,omitempty"`
	ProjectUUID      string     `json:"project_uuid,omitempty"`
	QueuedAt         time.Time  `json:"queued_at,omitempty"`
	Ref              string     `json:"ref,omitempty"`
	Status           string     `json:"status,omitempty"`
	Username         string     `json:"username,omitempty"`
	UUID             string     `json:"uuid,omitempty"`
}

// BuildList holds a list of Build objects
type BuildList struct {
	Builds []Build `json:"builds"`
	pagination
}

type buildResponse struct {
	Build Build `json:"build"`
}

// BuildPipelineMetrics structure of BuildPipelineMetrics object for a BuildPipeline
type BuildPipelineMetrics struct {
	AmiID                 string `json:"ami_id,omitempty"`
	Queries               string `json:"queries,omitempty"`
	CPUUser               string `json:"cpu_user,omitempty"`
	Duration              string `json:"duration,omitempty"`
	CPUSystem             string `json:"cpu_system,omitempty"`
	InstanceID            string `json:"instance_id,omitempty"`
	Architecture          string `json:"architecture,omitempty"`
	InstanceType          string `json:"instance_type,omitempty"`
	CPUPerSecond          string `json:"cpu_per_second,omitempty"`
	DiskFreeBytes         string `json:"disk_free_bytes,omitempty"`
	DiskUsedBytes         string `json:"disk_used_bytes,omitempty"`
	NetworkRxBytes        string `json:"network_rx_bytes,omitempty"`
	NetworkTxBytes        string `json:"network_tx_bytes,omitempty"`
	MaxUsedConnections    string `json:"max_used_connections,omitempty"`
	MemoryMaxUsageInBytes string `json:"memory_max_usage_in_bytes,omitempty"`
}

// BuildPipeline structure of BuildPipeline object for a Basic Project
type BuildPipeline struct {
	UUID       string               `json:"uuid,omitempty"`
	BuildUUID  string               `json:"build_uuid,omitempty"`
	Type       string               `json:"type,omitempty"`
	Status     string               `json:"status,omitempty"`
	CreatedAt  time.Time            `json:"created_at,omitempty"`
	UpdatedAt  time.Time            `json:"updated_at,omitempty"`
	FinishedAt time.Time            `json:"finished_at,omitempty"`
	Metrics    BuildPipelineMetrics `json:"metrics,omitempty"`
}

// BuildPipelines holds a list of BuildPipeline objects for a Basic Project
type BuildPipelines struct {
	Pipelines []BuildPipeline `json:"pipelines"`
	pagination
}

// BuildStep structure of BuildStep object for a Pro Project
type BuildStep struct {
	BuildUUID   string      `json:"build_uuid,omitempty"`
	BuildingAt  time.Time   `json:"building_at,omitempty"`
	Command     string      `json:"command,omitempty"`
	FinishedAt  time.Time   `json:"finished_at,omitempty"`
	ImageName   string      `json:"image_name,omitempty"`
	Name        string      `json:"name,omitempty"`
	Registry    string      `json:"registry,omitempty"`
	ServiceUUID string      `json:"service_uuid,omitempty"`
	StartedAt   time.Time   `json:"started_at,omitempty"`
	Status      string      `json:"status,omitempty"`
	Steps       []BuildStep `json:"steps,omitempty"`
	Tag         string      `json:"tag,omitempty"`
	Type        string      `json:"type,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	UUID        string      `json:"uuid,omitempty"`
}

// BuildSteps holds a list of BuildStep objects for a Pro Project
type BuildSteps struct {
	Steps []BuildStep `json:"steps"`
	pagination
}

// BuildService structure of BuildService object for a Pro Project
type BuildService struct {
	BuildUUID  string    `json:"build_uuid,omitempty"`
	BuildingAt time.Time `json:"building_at,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Name       string    `json:"name,omitempty"`
	PullingAt  time.Time `json:"pulling_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	UUID       string    `json:"uuid,omitempty"`
	Status     string    `json:"status,omitempty"`
}

// BuildServices holds a list of BuildService objects for a Pro Project
type BuildServices struct {
	Services []BuildService `json:"services"`
	pagination
}

type buildRequest struct {
	CommitSha string `json:"commit_sha,omitempty"`
	Ref       string `json:"ref,omitempty"`
}

// CreateBuild creates a new build
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/create-build
func (o *Organization) CreateBuild(ctx context.Context, projectUUID, ref, commitSha string) (bool, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s/builds", o.UUID, projectUUID)

	_, resp, err := o.client.request(ctx, "POST", path, buildRequest{
		Ref:       ref,
		CommitSha: commitSha,
	})
	if err != nil {
		return false, resp, errors.Wrap(err, "unable to create build")
	}

	return true, resp, nil
}

// GetBuild fetches a build by UUID
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/get-build
func (o *Organization) GetBuild(ctx context.Context, projectUUID, buildUUID string) (Build, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s/builds/%s", o.UUID, projectUUID, buildUUID)

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return Build{}, resp, errors.Wrap(err, "unable to get build")
	}

	var build buildResponse
	if err = json.Unmarshal(body, &build); err != nil {
		return Build{}, resp, errors.Wrap(err, "unable to unmarshal response into Build")
	}

	return build.Build, resp, nil
}

// ListBuilds fetches a list of builds
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/list-builds
func (o *Organization) ListBuilds(ctx context.Context, projectUUID string, opts ...PaginationOption) (BuildList, Response, error) {
	path, err := paginate(fmt.Sprintf("/organizations/%s/projects/%s/builds", o.UUID, projectUUID), opts...)
	if err != nil {
		return BuildList{}, Response{}, errors.Wrap(err, "unable to list builds")
	}

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return BuildList{}, resp, errors.Wrap(err, "unable to list builds")
	}

	var builds BuildList
	if err = json.Unmarshal(body, &builds); err != nil {
		return BuildList{}, resp, errors.Wrap(err, "unable to unmarshal response into BuildList")
	}

	return builds, resp, nil
}

// ListBuildPipelines lists Basic build pipelines
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/get-build-pipelines
func (o *Organization) ListBuildPipelines(ctx context.Context, projectUUID, buildUUID string, opts ...PaginationOption) (BuildPipelines, Response, error) {
	path, err := paginate(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s/pipelines", o.UUID, projectUUID, buildUUID), opts...)
	if err != nil {
		return BuildPipelines{}, Response{}, errors.Wrap(err, "unable to list pipelines")
	}

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return BuildPipelines{}, resp, errors.Wrap(err, "unable to list pipelines")
	}

	var pipelines BuildPipelines
	if err = json.Unmarshal(body, &pipelines); err != nil {
		return BuildPipelines{}, resp, errors.Wrap(err, "unable to unmarshal response into BuildPipelines")
	}

	return pipelines, resp, nil
}

// StopBuild stops a running build
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/stop-build
func (o *Organization) StopBuild(ctx context.Context, projectUUID, buildUUID string) (bool, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s/builds/%s/stop", o.UUID, projectUUID, buildUUID)

	_, resp, err := o.client.request(ctx, "POST", path, nil)
	if err != nil {
		return false, resp, errors.Wrap(err, "unable to stop build")
	}

	return true, resp, nil
}

// RestartBuild restarts a previous build
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/restart-build
func (o *Organization) RestartBuild(ctx context.Context, projectUUID, buildUUID string) (bool, Response, error) {
	path := fmt.Sprintf("/organizations/%s/projects/%s/builds/%s/restart", o.UUID, projectUUID, buildUUID)

	_, resp, err := o.client.request(ctx, "POST", path, nil)
	if err != nil {
		return false, resp, errors.Wrap(err, "unable to restart build")
	}

	return true, resp, nil
}

// ListBuildServices lists Pro build services
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/get-build-services
func (o *Organization) ListBuildServices(ctx context.Context, projectUUID, buildUUID string, opts ...PaginationOption) (BuildServices, Response, error) {
	path, err := paginate(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s/services", o.UUID, projectUUID, buildUUID), opts...)
	if err != nil {
		return BuildServices{}, Response{}, errors.Wrap(err, "unable to list build services")
	}

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return BuildServices{}, resp, errors.Wrap(err, "unable to list build services")
	}

	var services BuildServices
	if err = json.Unmarshal(body, &services); err != nil {
		return BuildServices{}, resp, errors.Wrap(err, "unable to unmarshal response into BuildServices")
	}

	return services, resp, nil
}

// ListBuildSteps lists Pro build steps
//
// Codeship API docs: https://apidocs.codeship.com/v2/builds/get-build-steps
func (o *Organization) ListBuildSteps(ctx context.Context, projectUUID, buildUUID string, opts ...PaginationOption) (BuildSteps, Response, error) {
	path, err := paginate(fmt.Sprintf("/organizations/%s/projects/%s/builds/%s/steps", o.UUID, projectUUID, buildUUID), opts...)
	if err != nil {
		return BuildSteps{}, Response{}, errors.Wrap(err, "unable to list build steps")
	}

	body, resp, err := o.client.request(ctx, "GET", path, nil)
	if err != nil {
		return BuildSteps{}, resp, errors.Wrap(err, "unable to list build steps")
	}

	var steps BuildSteps
	if err = json.Unmarshal(body, &steps); err != nil {
		return BuildSteps{}, resp, errors.Wrap(err, "unable to unmarshal response into BuildSteps")
	}

	return steps, resp, nil
}
