package syntax_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kpipelines/syntax"
	pipelinev1alpha1 "github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/knative/build-pipeline/test/builder"
	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Write a builder for generating the expected objects. Because
// as this is now, there are way too many lines here.
func TestParseJenkinsfileYaml(t *testing.T) {
	// Needed to take address of strings since workspace is *string. Is there a better way to handle optional values?
	defaultWorkspace := "default"
	customWorkspace := "custom"

	tests := []struct {
		name             string
		expected         *syntax.Jenkinsfile
		pipeline         *pipelinev1alpha1.Pipeline
		tasks            []*pipelinev1alpha1.Task
		expectedErrorMsg string
	}{
		{
			name: "simple_jenkinsfile",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "world"},
					}},
				}},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
			},
		},
		{
			name: "multiple_stages",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "A Working Stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"hello", "world"},
						}},
					},
					{
						Name: "Another stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"again"},
						}},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("a-working-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
				tb.Task("somepipeline-another-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("again")),
				)),
			},
		},
		{
			name: "nested_stages",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "Parent Stage",
						Stages: []syntax.Stage{
							{
								Name: "A Working Stage",
								Steps: []syntax.Step{{
									Command:   "echo",
									Arguments: []string{"hello", "world"},
								}},
							},
							{
								Name: "Another stage",
								Steps: []syntax.Step{{
									Command:   "echo",
									Arguments: []string{"again"},
								}},
							},
						},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("a-working-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
				tb.Task("somepipeline-another-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("again")),
				)),
			},
		},
		{
			name: "parallel_stages",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "First Stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"first"},
						}},
					},
					{
						Name: "Parent Stage",
						Parallel: []syntax.Stage{
							{
								Name: "A Working Stage",
								Steps: []syntax.Step{{
									Command:   "echo",
									Arguments: []string{"hello", "world"},
								}},
							},
							{
								Name: "Another stage",
								Steps: []syntax.Step{{
									Command:   "echo",
									Arguments: []string{"again"},
								}},
							},
						},
					},
					{
						Name: "Last Stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"last"},
						}},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage",
					// TODO: Switch from this kind of hackish approach to non-resource-based dependencies once they land.
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("a-working-stage", "another-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-first-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("first")),
				)),
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
				tb.Task("somepipeline-another-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("again")),
				)),
				tb.Task("somepipeline-last-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("last")),
				)),
			},
		},
		{
			name: "parallel_and_nested_stages",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "First Stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"first"},
						}},
					},
					{
						Name: "Parent Stage",
						Parallel: []syntax.Stage{
							{
								Name: "A Working Stage",
								Steps: []syntax.Step{{
									Command:   "echo",
									Arguments: []string{"hello", "world"},
								}},
							},
							{
								Name: "Nested In Parallel",
								Stages: []syntax.Stage{
									{
										Name: "Another stage",
										Steps: []syntax.Step{{
											Command:   "echo",
											Arguments: []string{"again"},
										}},
									},
									{
										Name: "Some other stage",
										Steps: []syntax.Step{{
											Command:   "echo",
											Arguments: []string{"otherwise"},
										}},
									},
								},
							},
						},
					},
					{
						Name: "Last Stage",
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"last"},
						}},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("some-other-stage", "somepipeline-some-other-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("another-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("another-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage",
					// TODO: Switch from this kind of hackish approach to non-resource-based dependencies once they land.
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("a-working-stage", "some-other-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-first-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("first")),
				)),
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
				tb.Task("somepipeline-another-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("again")),
				)),
				tb.Task("somepipeline-some-other-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("otherwise")),
				)),
				tb.Task("somepipeline-last-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("last")),
				)),
			},
		},
		{
			name: "custom_workspaces",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "stage1",
						Steps: []syntax.Step{{
							Command: "ls",
						}},
					},
					{
						Name: "stage2",
						Options: syntax.StageOptions{
							Workspace: &customWorkspace,
						},
						Steps: []syntax.Step{{
							Command: "ls",
						}},
					},
					{
						Name: "stage3",
						Options: syntax.StageOptions{
							Workspace: &defaultWorkspace,
						},
						Steps: []syntax.Step{{
							Command: "ls",
						}},
					},
					{
						Name: "stage4",
						Options: syntax.StageOptions{
							Workspace: &customWorkspace,
						},
						Steps: []syntax.Step{{
							Command: "ls",
						}},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("stage1", "somepipeline-stage1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage2", "somepipeline-stage2",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage1")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage3", "somepipeline-stage3",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("stage1")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage2")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage4", "somepipeline-stage4",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("stage2")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage3")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-stage1", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage2", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage3", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage4", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
			},
		},
		{
			name: "inherited_custom_workspaces",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: "stage1",
						Steps: []syntax.Step{{
							Command: "ls",
						}},
					},
					{
						Name: "stage2",
						Options: syntax.StageOptions{
							Workspace: &customWorkspace,
						},
						Stages: []syntax.Stage{
							{
								Name: "stage3",
								Steps: []syntax.Step{{
									Command: "ls",
								}},
							},
							{
								Name: "stage4",
								Options: syntax.StageOptions{
									Workspace: &defaultWorkspace,
								},
								Steps: []syntax.Step{{
									Command: "ls",
								}},
							},
							{
								Name: "stage5",
								Steps: []syntax.Step{{
									Command: "ls",
								}},
							},
						},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("stage1", "somepipeline-stage1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage3", "somepipeline-stage3",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage1")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage4", "somepipeline-stage4",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("stage1")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage3")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("stage5", "somepipeline-stage5",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("stage3")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From("stage4")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-stage1", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage3", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage4", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-stage5", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
			},
		},
		{
			name: "environment_at_top_and_in_stage",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Environment: []syntax.EnvVar{{
					Name:  "SOME_VAR",
					Value: "A value for the env var",
				}},
				Stages: []syntax.Stage{{
					Name: "A stage with environment",
					Environment: []syntax.EnvVar{{
						Name:  "SOME_OTHER_VAR",
						Value: "A value for the other env var",
					}},
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "${SOME_OTHER_VAR}"},
					}},
				}},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-stage-with-environment", "somepipeline-a-stage-with-environment",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-stage-with-environment", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "${SOME_OTHER_VAR}"),
						// TODO: Ordering doesn't seem to be deterministic.
						tb.EnvVar("SOME_VAR", "A value for the env var"), tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var")),
				)),
			},
		},
		{
			name: "syntactic_sugar_step_and_a_command",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Steps: []syntax.Step{
						{
							Command:   "echo",
							Arguments: []string{"hello", "world"},
						},
						{
							Step: "some-step",
							Options: map[string]string{
								"firstParam":  "some value",
								"secondParam": "some other value",
							},
						},
					},
				}},
			},
			expectedErrorMsg: "syntactic sugar steps not yet supported",
		},
		{
			name: "post",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "world"},
					}},
					Post: []syntax.Post{
						{
							Condition: "success",
							Actions: []syntax.PostAction{{
								Name: "mail",
								Options: map[string]string{
									"to":      "foo@bar.com",
									"subject": "Yay, it passed",
								},
							}},
						},
						{
							Condition: "failure",
							Actions: []syntax.PostAction{{
								Name: "slack",
								Options: map[string]string{
									"whatever": "the",
									"slack":    "config",
									"actually": "is. =)",
								},
							}},
						},
						{
							Condition: "always",
							Actions: []syntax.PostAction{{
								Name: "junit",
								Options: map[string]string{
									"pattern": "target/surefire-reports/**/*.xml",
								},
							}},
						},
					},
				}},
			},
			expectedErrorMsg: "post on stages not yet supported",
		},
		{
			name: "top_level_and_stage_options",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Options: syntax.RootOptions{
					Timeout: syntax.Timeout{
						Time: 50,
						Unit: "minutes",
					},
					Retry: 3,
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Options: syntax.StageOptions{
						RootOptions: syntax.RootOptions{
							Timeout: syntax.Timeout{
								Time: 5,
								Unit: "seconds",
							},
							Retry: 4,
						},
						Stash: syntax.Stash{
							Name:  "Some Files",
							Files: "somedir/**/*",
						},
						Unstash: syntax.Unstash{
							Name: "Earlier Files",
							Dir:  "some/sub/dir",
						},
					},
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "world"},
					}},
				}},
			},
			expectedErrorMsg: "Retry at top level not yet supported",
		},
		{
			name: "stage_and_step_agent",
			expected: &syntax.Jenkinsfile{
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Agent: syntax.Agent{
						Image: "some-image",
					},
					Steps: []syntax.Step{
						{
							Command:   "echo",
							Arguments: []string{"hello", "world"},
							Agent: syntax.Agent{
								Image: "some-other-image",
							},
						},
						{
							Command:   "echo",
							Arguments: []string{"goodbye"},
						},
					},
				}},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-other-image", tb.Command("echo"), tb.Args("hello", "world")),
					tb.Step("step3", "some-image", tb.Command("echo"), tb.Args("goodbye")),
				)),
			},
		},
		{
			name: "mangled_task_names",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{
					{
						Name: ". -a- .",
						Steps: []syntax.Step{{
							Command:   "ls",
							Arguments: nil,
						}},
					},
					{
						Name: "Wööh!!!! - This is cool.",
						Steps: []syntax.Step{{
							Command:   "ls",
							Arguments: nil,
						}},
					},
				},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask(".--a--.", "somepipeline-a",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineTask("wööh!!!!---this-is-cool.", "somepipeline-wh-this-is-cool",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From(".--a--.")),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource",
						tb.From(".--a--.")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
				tb.Task("somepipeline-wh-this-is-cool", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("ls")),
				)),
			},
		},
		{
			name: "stage_timeout",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Options: syntax.StageOptions{
						RootOptions: syntax.RootOptions{
							Timeout: syntax.Timeout{
								Time: 50,
								Unit: "minutes",
							},
						},
					},
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "world"},
					}},
				}},
			},
			/* TODO: Stop erroring out once we figure out how to handle task timeouts again
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskTimeout(50*time.Minute),
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
			},*/
			expectedErrorMsg: "Timeout on stage not yet supported",
		},
		{
			name: "top_level_timeout",
			expected: &syntax.Jenkinsfile{
				Agent: syntax.Agent{
					Image: "some-image",
				},
				Options: syntax.RootOptions{
					Timeout: syntax.Timeout{
						Time: 50,
						Unit: "minutes",
					},
				},
				Stages: []syntax.Stage{{
					Name: "A Working Stage",
					Steps: []syntax.Step{{
						Command:   "echo",
						Arguments: []string{"hello", "world"},
					}},
				}},
			},
			pipeline: tb.Pipeline("somepipeline", "somenamespace", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("temp-ordering-resource", "temp-ordering-resource")),
				tb.PipelineDeclaredResource("somepipeline", pipelinev1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage))),
			tasks: []*pipelinev1alpha1.Task{
				tb.Task("somepipeline-a-working-stage", "somenamespace", tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace")),
						tb.InputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.TaskOutputs(tb.OutputsResource("workspace", pipelinev1alpha1.PipelineResourceTypeGit),
						tb.OutputsResource("temp-ordering-resource", pipelinev1alpha1.PipelineResourceTypeImage)),
					tb.Step("step2", "some-image", tb.Command("echo"), tb.Args("hello", "world")),
				)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConfig, fn, err := config.LoadProjectConfig("test_data/" + tt.name)
			if err != nil {
				t.Fatalf("Failed to parse YAML for %s: %q", tt.name, err)
			}

			if projectConfig.PipelineConfig == nil {
				t.Fatalf("PipelineConfig at %s is nil: %+v", fn, projectConfig)
			}
			if &projectConfig.PipelineConfig.Pipelines == nil {
				t.Fatalf("Pipelines at %s is nil: %+v", fn, projectConfig.PipelineConfig)
			}
			if projectConfig.PipelineConfig.Pipelines.Release == nil {
				t.Fatalf("Release at %s is nil: %+v", fn, projectConfig.PipelineConfig.Pipelines)
			}
			if projectConfig.PipelineConfig.Pipelines.Release.Pipeline == nil {
				t.Fatalf("Pipeline at %s is nil: %+v", fn, projectConfig.PipelineConfig.Pipelines.Release)
			}
			parsed := projectConfig.PipelineConfig.Pipelines.Release.Pipeline

			if d := cmp.Diff(tt.expected, parsed); d != "" {
				t.Errorf("Parsed Jenkinsfile did not match expected: %s", d)
			}

			validateErr := parsed.Validate()
			if validateErr != nil {
				t.Errorf("Validation failed: %s", validateErr)
			}

			pipeline, tasks, err := parsed.GenerateCRDs("somepipeline", "somebuild", "somenamespace", "abcd", nil)

			if err != nil {
				if tt.expectedErrorMsg != "" {
					if d := cmp.Diff(tt.expectedErrorMsg, err.Error()); d != "" {
						t.Fatalf("CRD generation error did not meet expectation: %s", d)
					}
				} else {
					t.Fatalf("Error generating CRDs: %s", err)
				}
			}

			if tt.expectedErrorMsg == "" {
				pipeline.TypeMeta = metav1.TypeMeta{}
				if d := cmp.Diff(tt.pipeline, pipeline); d != "" {
					t.Errorf("Generated Pipeline did not match expected: %s", d)
				}

				if err := pipeline.Spec.Validate(); err != nil {
					t.Errorf("PipelineSpec.Validate() = %v", err)
				}

				for _, task := range tasks {
					task.TypeMeta = metav1.TypeMeta{}
				}
				if d := cmp.Diff(tt.tasks, tasks); d != "" {
					t.Errorf("Generated Tasks did not match expected: %s", d)
				}

				for _, task := range tasks {
					if err := task.Spec.Validate(); err != nil {
						t.Errorf("TaskSpec.Validate() = %v", err)
					}
				}
			}
		})
	}
}

func TestFailedValidation(t *testing.T) {
	tests := []struct {
		name          string
		expectedError *apis.FieldError
	}{
		/* TODO: Once we figure out how to differentiate between an empty agent and no agent specified...
		{
			name: "empty_agent",
			expectedError: &apis.FieldError{
				Message: "Invalid apiVersion format: must be 'v(digits).(digits)",
				Paths:   []string{"apiVersion"},
			},
		},
		*/
		{
			name: "agent_with_both_image_and_label",
			expectedError: apis.ErrMultipleOneOf("label", "image").
				ViaField("agent"),
		},
		{
			name:          "no_stages",
			expectedError: apis.ErrMissingField("stages"),
		},
		{
			name:          "no_steps_stages_or_parallel",
			expectedError: apis.ErrMissingOneOf("steps", "stages", "parallel").ViaFieldIndex("stages", 0),
		},
		{
			name:          "steps_and_stages",
			expectedError: apis.ErrMultipleOneOf("steps", "stages", "parallel").ViaFieldIndex("stages", 0),
		},
		{
			name:          "steps_and_parallel",
			expectedError: apis.ErrMultipleOneOf("steps", "stages", "parallel").ViaFieldIndex("stages", 0),
		},
		{
			name:          "stages_and_parallel",
			expectedError: apis.ErrMultipleOneOf("steps", "stages", "parallel").ViaFieldIndex("stages", 0),
		},
		{
			name:          "step_without_command_or_step",
			expectedError: apis.ErrMissingOneOf("command", "step").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name:          "step_with_both_command_and_step",
			expectedError: apis.ErrMultipleOneOf("command", "step").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_command_and_options",
			expectedError: (&apis.FieldError{
				Message: "Cannot set options for a command",
				Paths:   []string{"options"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_step_and_arguments",
			expectedError: (&apis.FieldError{
				Message: "Cannot set command-line arguments for a step",
				Paths:   []string{"args"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "no_parent_or_stage_agent",
			expectedError: (&apis.FieldError{
				Message: "No agent specified for stage or for its parent(s)",
				Paths:   []string{"agent"},
			}).ViaFieldIndex("stages", 0),
		},
		{
			name: "top_level_timeout_without_time",
			expectedError: (&apis.FieldError{
				Message: "Timeout must be greater than zero",
				Paths:   []string{"time"},
			}).ViaField("timeout").ViaField("options"),
		},
		{
			name: "stage_timeout_without_time",
			expectedError: (&apis.FieldError{
				Message: "Timeout must be greater than zero",
				Paths:   []string{"time"},
			}).ViaField("timeout").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "top_level_timeout_with_invalid_unit",
			expectedError: (&apis.FieldError{
				Message: "years is not a valid time unit. Valid time units are seconds, minutes, hours, days",
				Paths:   []string{"unit"},
			}).ViaField("timeout").ViaField("options"),
		},
		{
			name: "stage_timeout_with_invalid_unit",
			expectedError: (&apis.FieldError{
				Message: "years is not a valid time unit. Valid time units are seconds, minutes, hours, days",
				Paths:   []string{"unit"},
			}).ViaField("timeout").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "top_level_timeout_with_invalid_time",
			expectedError: (&apis.FieldError{
				Message: "Timeout must be greater than zero",
				Paths:   []string{"time"},
			}).ViaField("timeout").ViaField("options"),
		},
		{
			name: "stage_timeout_with_invalid_time",
			expectedError: (&apis.FieldError{
				Message: "Timeout must be greater than zero",
				Paths:   []string{"time"},
			}).ViaField("timeout").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "top_level_retry_with_invalid_count",
			expectedError: (&apis.FieldError{
				Message: "Retry count cannot be negative",
				Paths:   []string{"retry"},
			}).ViaField("options"),
		},
		{
			name: "stage_retry_with_invalid_count",
			expectedError: (&apis.FieldError{
				Message: "Retry count cannot be negative",
				Paths:   []string{"retry"},
			}).ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "stash_without_name",
			expectedError: (&apis.FieldError{
				Message: "The stash name must be provided",
				Paths:   []string{"name"},
			}).ViaField("stash").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "stash_without_files",
			expectedError: (&apis.FieldError{
				Message: "files to stash must be provided",
				Paths:   []string{"files"},
			}).ViaField("stash").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "unstash_without_name",
			expectedError: (&apis.FieldError{
				Message: "The unstash name must be provided",
				Paths:   []string{"name"},
			}).ViaField("unstash").ViaField("options").ViaFieldIndex("stages", 0),
		},
		{
			name: "blank_stage_name",
			expectedError: (&apis.FieldError{
				Message: "Stage name must contain at least one ASCII letter",
				Paths:   []string{"name"},
			}).ViaFieldIndex("stages", 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlFile := filepath.Join("test_data", "validation_failures", tt.name+".yaml")
			YamlToRead, YamlReadErr := ioutil.ReadFile(yamlFile)
			if YamlReadErr != nil {
				t.Fatalf("Could not read yaml file: %s ", yamlFile)
			}
			tt.name = string(YamlToRead)

			parsed, parseErr := syntax.ParseJenkinsfileYaml(tt.name)
			if parseErr != nil {
				t.Fatalf("Failed to parse YAML for %s: %q", tt.name, parseErr)
			}

			err := parsed.Validate()

			if err == nil {
				t.Fatalf("Expected a validation failure but none occurred")
			}

			if d := cmp.Diff(tt.expectedError, err, cmp.AllowUnexported(apis.FieldError{})); d != "" {
				t.Fatalf("Validation error did not meet expectation: %s", d)
			}
		})
	}
}

func TestRfc1035LabelMangling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unmodified",
			input:    "unmodified",
			expected: "unmodified-suffix",
		},
		{
			name:     "spaces",
			input:    "A Simple Test.",
			expected: "a-simple-test-suffix",
		},
		{
			name:     "no leading digits",
			input:    "0123456789no-leading-digits",
			expected: "no-leading-digits-suffix",
		},
		{
			name:     "no leading hyphens",
			input:    "----no-leading-hyphens",
			expected: "no-leading-hyphens-suffix",
		},
		{
			name:     "no consecutive hyphens",
			input:    "no--consecutive- hyphens",
			expected: "no-consecutive-hyphens-suffix",
		},
		{
			name:     "no trailing hyphens",
			input:    "no-trailing-hyphens----",
			expected: "no-trailing-hyphens-suffix",
		},
		{
			name:     "no symbols",
			input:    "&$^#@(*&$^-whoops",
			expected: "whoops-suffix",
		},
		{
			name:     "no unprintable characters",
			input:    "a\n\t\x00b",
			expected: "ab-suffix",
		},
		{
			name:     "no unicode",
			input:    "japan-日本",
			expected: "japan-suffix",
		},
		{
			name:     "no non-bmp characters",
			input:    "happy 😃",
			expected: "happy-suffix",
		},
		{
			name:     "truncated to 63",
			input:    "a0123456789012345678901234567890123456789012345678901234567890123456789",
			expected: "a0123456789012345678901234567890123456789012345678901234-suffix",
		},
		{
			name:     "truncated to 62",
			input:    "a012345678901234567890123456789012345678901234567890123-567890123456789",
			expected: "a012345678901234567890123456789012345678901234567890123-suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mangled := syntax.MangleToRfc1035Label(tt.input, "suffix")
			if d := cmp.Diff(tt.expected, mangled); d != "" {
				t.Fatalf("Mangled output did not match expected output: %s", d)
			}
		})
	}
}
