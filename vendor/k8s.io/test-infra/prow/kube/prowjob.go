/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kube

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProwJobType specifies how the job is triggered.
type ProwJobType string

// ProwJobState specifies whether the job is running
type ProwJobState string

// ProwJobAgent specifies the controller (such as plank or jenkins-agent) that runs the job.
type ProwJobAgent string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob ProwJobType = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob = "batch"
)

// Various job states.
const (
	// TriggeredState means the job has been created but not yet scheduled.
	TriggeredState ProwJobState = "triggered"
	// PendingState means the job is scheduled but not yet running.
	PendingState = "pending"
	// SuccessState means the job completed without error (exit 0)
	SuccessState = "success"
	// FailureState means the job completed with errors (exit non-zero)
	FailureState = "failure"
	// AbortedState means prow killed the job early (new commit pushed, perhaps).
	AbortedState = "aborted"
	// ErrorState means the job could not schedule (bad config, perhaps).
	ErrorState = "error"
)

const (
	// KubernetesAgent means prow will create a pod to run this job.
	KubernetesAgent ProwJobAgent = "kubernetes"
	// JenkinsAgent means prow will schedule the job on jenkins.
	JenkinsAgent = "jenkins"
)

const (
	// CreatedByProw is added on pods created by prow. We cannot
	// really use owner references because pods may reside on a
	// different namespace from the namespace parent prowjobs
	// live and that would cause the k8s garbage collector to
	// identify those prow pods as orphans and delete them
	// instantly.
	// TODO: Namespace this label.
	CreatedByProw = "created-by-prow"
	// ProwJobTypeLabel is added in pods created by prow and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	ProwJobTypeLabel = "prow.k8s.io/type"
	// ProwJobIDLabel is added in pods created by prow and
	// carries the ID of the ProwJob that the pod is fulfilling.
	// We also name pods after the ProwJob that spawned them but
	// this allows for multiple resources to be linked to one
	// ProwJob.
	ProwJobIDLabel = "prow.k8s.io/id"
	// ProwJobAnnotation is added in pods created by prow and
	// carries the name of the job that the pod is running. Since
	// job names can be arbitrarily long, this is added as
	// an annotation instead of a label.
	ProwJobAnnotation = "prow.k8s.io/job"
)

// ProwJob contains the spec as well as runtime metadata.
type ProwJob struct {
	APIVersion        string `json:"apiVersion,omitempty"`
	Kind              string `json:"kind,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProwJobSpec   `json:"spec,omitempty"`
	Status ProwJobStatus `json:"status,omitempty"`
}

// ProwJobSpec configures the details of the prow job.
//
// Details include the podspec, code to clone, the cluster it runs
// any child jobs, concurrency limitations, etc.
type ProwJobSpec struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type ProwJobType `json:"type,omitempty"`
	// Agent determines which controller fulfills
	// this specific ProwJobSpec and runs the job
	Agent ProwJobAgent `json:"agent,omitempty"`
	// Cluster is which Kubernetes cluster is used
	// to run the job, only applicable for that
	// specific agent
	Cluster string `json:"cluster,omitempty"`
	// Job is the name of the job
	Job string `json:"job,omitempty"`
	// Refs is the code under test, determined at
	// runtime by Prow itself
	Refs *Refs `json:"refs,omitempty"`
	// ExtraRefs are auxiliary repositories that
	// need to be cloned, determined from config
	ExtraRefs []*Refs `json:"extra_refs,omitempty"`

	// Report determines if the result of this job should
	// be posted as a status on GitHub
	Report bool `json:"report,omitempty"`
	// Context is the name of the status context used to
	// report back to GitHub
	Context string `json:"context,omitempty"`
	// RerunCommand is the command a user would write to
	// trigger this job on their pull request
	RerunCommand string `json:"rerun_command,omitempty"`
	// MaxConcurrency restricts the total number of instances
	// of this job that can run in parallel at once
	MaxConcurrency int `json:"max_concurrency,omitempty"`

	// PodSpec provides the basis for running the test under
	// a Kubernetes agent
	PodSpec *v1.PodSpec `json:"pod_spec,omitempty"`

	// DecorationConfig holds configuration options for
	// decorating PodSpecs that users provide
	DecorationConfig *DecorationConfig `json:"decoration_config,omitempty"`

	// RunAfterSuccess are jobs that should be triggered if
	// this job runs and does not fail
	RunAfterSuccess []ProwJobSpec `json:"run_after_success,omitempty"`
}

// DecorationConfig specifies how to augment pods.
//
// This is primarily used to provide automatic integration with gubernator
// and testgrid.
type DecorationConfig struct {
	// Timeout is how long the pod utilities will wait
	// before aborting a job with SIGINT.
	Timeout time.Duration `json:"timeout,omitempty"`
	// GracePeriod is how long the pod utilities will wait
	// after sending SIGINT to send SIGKILL when aborting
	// a job. Only applicable if decorating the PodSpec.
	GracePeriod time.Duration `json:"grace_period,omitempty"`
	// UtilityImages holds pull specs for utility container
	// images used to decorate a PodSpec.
	UtilityImages *UtilityImages `json:"utility_images,omitempty"`
	// GCSConfiguration holds options for pushing logs and
	// artifacts to GCS from a job.
	GCSConfiguration *GCSConfiguration `json:"gcs_configuration,omitempty"`
	// GCSCredentialsSecret is the name of the Kubernetes secret
	// that holds GCS push credentials
	GCSCredentialsSecret string `json:"gcs_credentials_secret,omitempty"`
	// SSHKeySecrets are the names of Kubernetes secrets that contain
	// SSK keys which should be used during the cloning process
	SSHKeySecrets []string `json:"ssh_key_secrets,omitempty"`
	// SkipCloning determines if we should clone source code in the
	// initcontainers for jobs that specify refs
	SkipCloning bool `json:"skip_cloning,omitempty"`
}

// UtilityImages holds pull specs for the utility images
// to be used for a job
type UtilityImages struct {
	// CloneRefs is the pull spec used for the clonerefs utility
	CloneRefs string `json:"clonerefs,omitempty"`
	// InitUpload is the pull spec used for the initupload utility
	InitUpload string `json:"initupload,omitempty"`
	// Entrypoint is the pull spec used for the entrypoint utility
	Entrypoint string `json:"entrypoint,omitempty"`
	// sidecar is the pull spec used for the sidecar utility
	Sidecar string `json:"sidecar,omitempty"`
}

// PathStrategy specifies minutia about how to contruct the url.
// Usually consumed by gubernator/testgrid.
const (
	PathStrategyLegacy   = "legacy"
	PathStrategySingle   = "single"
	PathStrategyExplicit = "explicit"
)

// GCSConfiguration holds options for pushing logs and
// artifacts to GCS from a job.
type GCSConfiguration struct {
	// Bucket is the GCS bucket to upload to
	Bucket string `json:"bucket,omitempty"`
	// PathPrefix is an optional path that follows the
	// bucket name and comes before any structure
	PathPrefix string `json:"path_prefix,omitempty"`
	// PathStrategy dictates how the org and repo are used
	// when calculating the full path to an artifact in GCS
	PathStrategy string `json:"path_strategy,omitempty"`
	// DefaultOrg is omitted from GCS paths when using the
	// legacy or simple strategy
	DefaultOrg string `json:"default_org,omitempty"`
	// DefaultRepo is omitted from GCS paths when using the
	// legacy or simple strategy
	DefaultRepo string `json:"default_repo,omitempty"`
}

// ProwJobStatus provides runtime metadata, such as when it finished, whether it is running, etc.
type ProwJobStatus struct {
	StartTime      metav1.Time  `json:"startTime,omitempty"`
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	State          ProwJobState `json:"state,omitempty"`
	Description    string       `json:"description,omitempty"`
	URL            string       `json:"url,omitempty"`

	// PodName applies only to ProwJobs fulfilled by
	// plank. This field should always be the same as
	// the ProwJob.ObjectMeta.Name field.
	PodName string `json:"pod_name,omitempty"`

	// BuildID is the build identifier vended either by tot
	// or the snowflake library for this job and used as an
	// identifier for grouping artifacts in GCS for views in
	// TestGrid and Gubernator. Idenitifiers vended by tot
	// are monotonically increasing whereas identifiers vended
	// by the snowflake library are not.
	BuildID string `json:"build_id,omitempty"`

	// JenkinsBuildID applies only to ProwJobs fulfilled
	// by the jenkins-operator. This field is the build
	// identifier that Jenkins gave to the build for this
	// ProwJob.
	JenkinsBuildID string `json:"jenkins_build_id,omitempty"`
}

// Complete returns true if the prow job has finished
func (j *ProwJob) Complete() bool {
	// TODO(fejta): support a timeout?
	return j.Status.CompletionTime != nil
}

// SetComplete marks the job as completed (at time now).
func (j *ProwJob) SetComplete() {
	j.Status.CompletionTime = new(metav1.Time)
	*j.Status.CompletionTime = metav1.Now()
}

// ClusterAlias specifies the key in the clusters map to use.
//
// This allows scheduling a prow job somewhere aside from the default build cluster.
func (j *ProwJob) ClusterAlias() string {
	if j.Spec.Cluster == "" {
		return DefaultClusterAlias
	}
	return j.Spec.Cluster
}

// Pull describes a pull request at a particular point in time.
type Pull struct {
	Number int    `json:"number,omitempty"`
	Author string `json:"author,omitempty"`
	SHA    string `json:"sha,omitempty"`

	// Ref is git ref can be checked out for a change
	// for example,
	// github: pull/123/head
	// gerrit: refs/changes/00/123/1
	Ref string `json:"ref,omitempty"`
}

// Refs describes how the repo was constructed.
type Refs struct {
	// Org is something like kubernetes or k8s.io
	Org string `json:"org,omitempty"`
	// Repo is something like test-infra
	Repo string `json:"repo,omitempty"`

	BaseRef string `json:"base_ref,omitempty"`
	BaseSHA string `json:"base_sha,omitempty"`

	Pulls []Pull `json:"pulls,omitempty"`

	// PathAlias is the location under <root-dir>/src
	// where this repository is cloned. If this is not
	// set, <root-dir>/src/github.com/org/repo will be
	// used as the default.
	PathAlias string `json:"path_alias,omitempty"`
	// CloneURI is the URI that is used to clone the
	// repository. If unset, will default to
	// `https://github.com/org/repo.git`.
	CloneURI string `json:"clone_uri,omitempty"`
}

func (r Refs) String() string {
	rs := []string{fmt.Sprintf("%s:%s", r.BaseRef, r.BaseSHA)}
	for _, pull := range r.Pulls {
		ref := fmt.Sprintf("%d:%s", pull.Number, pull.SHA)

		if pull.Ref != "" {
			ref = fmt.Sprintf("%s:%s", ref, pull.Ref)
		}

		rs = append(rs, ref)
	}
	return strings.Join(rs, ",")
}
