// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package k8s_io_api_batch_v1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"k8s.io/api/batch/v1.Job":          schema_k8sio_api_batch_v1_Job(ref),
		"k8s.io/api/batch/v1.JobCondition": schema_k8sio_api_batch_v1_JobCondition(ref),
		"k8s.io/api/batch/v1.JobList":      schema_k8sio_api_batch_v1_JobList(ref),
		"k8s.io/api/batch/v1.JobSpec":      schema_k8sio_api_batch_v1_JobSpec(ref),
		"k8s.io/api/batch/v1.JobStatus":    schema_k8sio_api_batch_v1_JobStatus(ref),
	}
}

func schema_k8sio_api_batch_v1_Job(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Job represents the configuration of a single job.",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Description: "Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Description: "Specification of the desired behavior of a job. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status",
							Ref:         ref("k8s.io/api/batch/v1.JobSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Description: "Current status of a job. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status",
							Ref:         ref("k8s.io/api/batch/v1.JobStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/api/batch/v1.JobSpec", "k8s.io/api/batch/v1.JobStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_k8sio_api_batch_v1_JobCondition(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "JobCondition describes current state of a job.",
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Description: "Type of job condition, Complete or Failed.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Description: "Status of the condition, one of True, False, Unknown.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"lastProbeTime": {
						SchemaProps: spec.SchemaProps{
							Description: "Last time the condition was checked.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"lastTransitionTime": {
						SchemaProps: spec.SchemaProps{
							Description: "Last time the condition transit from one status to another.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"reason": {
						SchemaProps: spec.SchemaProps{
							Description: "(brief) reason for the condition's last transition.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"message": {
						SchemaProps: spec.SchemaProps{
							Description: "Human readable message indicating details about last transition.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
				Required: []string{"type", "status"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Time"},
	}
}

func schema_k8sio_api_batch_v1_JobList(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "JobList is a collection of jobs.",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Description: "Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"),
						},
					},
					"items": {
						SchemaProps: spec.SchemaProps{
							Description: "items is the list of Jobs.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("k8s.io/api/batch/v1.Job"),
									},
								},
							},
						},
					},
				},
				Required: []string{"items"},
			},
		},
		Dependencies: []string{
			"k8s.io/api/batch/v1.Job", "k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"},
	}
}

func schema_k8sio_api_batch_v1_JobSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "JobSpec describes how the job execution will look like.",
				Properties: map[string]spec.Schema{
					"parallelism": {
						SchemaProps: spec.SchemaProps{
							Description: "Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"completions": {
						SchemaProps: spec.SchemaProps{
							Description: "Specifies the desired number of successfully finished pods the job should be run with.  Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value.  Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"activeDeadlineSeconds": {
						SchemaProps: spec.SchemaProps{
							Description: "Specifies the duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer",
							Type:        []string{"integer"},
							Format:      "int64",
						},
					},
					"backoffLimit": {
						SchemaProps: spec.SchemaProps{
							Description: "Specifies the number of retries before marking this job failed. Defaults to 6",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"selector": {
						SchemaProps: spec.SchemaProps{
							Description: "A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"manualSelector": {
						SchemaProps: spec.SchemaProps{
							Description: "manualSelector controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template.  When true, the user is responsible for picking unique labels and specifying the selector.  Failure to pick a unique label may cause this and other jobs to not function correctly.  However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector",
							Type:        []string{"boolean"},
							Format:      "",
						},
					},
					"template": {
						SchemaProps: spec.SchemaProps{
							Description: "Describes the pod that will be created when executing a job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
							Ref:         ref("k8s.io/api/core/v1.PodTemplateSpec"),
						},
					},
					"ttlSecondsAfterFinished": {
						SchemaProps: spec.SchemaProps{
							Description: "ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won't be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes. This field is alpha-level and is only honored by servers that enable the TTLAfterFinished feature.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
				},
				Required: []string{"template"},
			},
		},
		Dependencies: []string{
			"k8s.io/api/core/v1.PodTemplateSpec", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"},
	}
}

func schema_k8sio_api_batch_v1_JobStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "JobStatus represents the current state of a Job.",
				Properties: map[string]spec.Schema{
					"conditions": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-patch-merge-key": "type",
								"x-kubernetes-patch-strategy":  "merge",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "The latest available observations of an object's current state. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("k8s.io/api/batch/v1.JobCondition"),
									},
								},
							},
						},
					},
					"startTime": {
						SchemaProps: spec.SchemaProps{
							Description: "Represents time when the job was acknowledged by the job controller. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"completionTime": {
						SchemaProps: spec.SchemaProps{
							Description: "Represents time when the job was completed. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"active": {
						SchemaProps: spec.SchemaProps{
							Description: "The number of actively running pods.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"succeeded": {
						SchemaProps: spec.SchemaProps{
							Description: "The number of pods which reached phase Succeeded.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"failed": {
						SchemaProps: spec.SchemaProps{
							Description: "The number of pods which reached phase Failed.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/api/batch/v1.JobCondition", "k8s.io/apimachinery/pkg/apis/meta/v1.Time"},
	}
}
