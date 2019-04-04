package syntax_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/kmp"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Needed to take address of strings since workspace is *string. Is there a better way to handle optional values?
	defaultWorkspace = "default"
	customWorkspace  = "custom"
)

// TODO: Try to write some helper functions to make Pipeline and Task expect building less bloody verbose.
func TestParseJenkinsfileYaml(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name               string
		expected           *syntax.ParsedPipeline
		pipeline           *tektonv1alpha1.Pipeline
		tasks              []*tektonv1alpha1.Task
		expectedErrorMsg   string
		validationErrorMsg string
		structure          *v1.PipelineStructure
	}{
		{
			name: "simple_jenkinsfile",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "multiple_stages",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world")),
				),
				PipelineStage("Another stage",
					StageStep(
						StepCmd("echo"),
						StepArg("again"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("a-working-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-another-stage-1", "jx", TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)), tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
				StructureStage("Another stage", StructureStageTaskRef("somepipeline-another-stage-1"),
					StructureStagePrevious("A Working Stage")),
			),
		},
		{
			name: "nested_stages",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("Parent Stage",
					StageSequential("A Working Stage",
						StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world"))),
					StageSequential("Another stage",
						StageStep(StepCmd("echo"), StepArg("again"))),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("a-working-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit, tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("Parent Stage",
					StructureStageStages("A Working Stage", "Another stage")),
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1"),
					StructureStageDepth(1),
					StructureStageParent("Parent Stage")),
				StructureStage("Another stage", StructureStageTaskRef("somepipeline-another-stage-1"),
					StructureStageDepth(1),
					StructureStageParent("Parent Stage"),
					StructureStagePrevious("A Working Stage")),
			),
		},
		{
			name: "parallel_stages",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("First Stage",
					StageStep(StepCmd("echo"), StepArg("first"))),
				PipelineStage("Parent Stage",
					StageParallel("A Working Stage",
						StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world"))),
					StageParallel("Another stage",
						StageStep(StepCmd("echo"), StepArg("again"))),
				),
				PipelineStage("Last Stage",
					StageStep(StepCmd("echo"), StepArg("last"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")), tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("a-working-stage", "another-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-first-stage-1", "jx", TaskStageLabel("First Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo first"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-last-stage-1", "jx", TaskStageLabel("Last Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)), tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo last"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("First Stage", StructureStageTaskRef("somepipeline-first-stage-1")),
				StructureStage("Parent Stage",
					StructureStageParallel("A Working Stage", "Another stage"),
					StructureStagePrevious("First Stage"),
				),
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1"),
					StructureStageDepth(1),
					StructureStageParent("Parent Stage"),
				),
				StructureStage("Another stage", StructureStageTaskRef("somepipeline-another-stage-1"),
					StructureStageDepth(1),
					StructureStageParent("Parent Stage"),
				),
				StructureStage("Last Stage", StructureStageTaskRef("somepipeline-last-stage-1"),
					StructureStagePrevious("Parent Stage")),
			),
		},
		{
			name: "parallel_and_nested_stages",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("First Stage",
					StageStep(StepCmd("echo"), StepArg("first"))),
				PipelineStage("Parent Stage",
					StageParallel("A Working Stage",
						StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world"))),
					StageParallel("Nested In Parallel",
						StageSequential("Another stage",
							StageStep(StepCmd("echo"), StepArg("again"))),
						StageSequential("Some other stage",
							StageStep(StepCmd("echo"), StepArg("otherwise"))),
					),
				),
				PipelineStage("Last Stage",
					StageStep(StepCmd("echo"), StepArg("last"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("some-other-stage", "somepipeline-some-other-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("another-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("another-stage")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("a-working-stage", "some-other-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-first-stage-1", "jx", TaskStageLabel("First Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo first"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-some-other-stage-1", "jx", TaskStageLabel("Some other stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo otherwise"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-last-stage-1", "jx", TaskStageLabel("Last Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo last"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("First Stage", StructureStageTaskRef("somepipeline-first-stage-1")),
				StructureStage("Parent Stage",
					StructureStageParallel("A Working Stage", "Nested In Parallel"),
					StructureStagePrevious("First Stage"),
				),
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1"),
					StructureStageDepth(1),
					StructureStageParent("Parent Stage"),
				),
				StructureStage("Nested In Parallel",
					StructureStageParent("Parent Stage"),
					StructureStageDepth(1),
					StructureStageStages("Another stage", "Some other stage"),
				),
				StructureStage("Another stage", StructureStageTaskRef("somepipeline-another-stage-1"),
					StructureStageDepth(2),
					StructureStageParent("Nested In Parallel"),
				),
				StructureStage("Some other stage", StructureStageTaskRef("somepipeline-some-other-stage-1"),
					StructureStageDepth(2),
					StructureStageParent("Nested In Parallel"),
					StructureStagePrevious("Another stage"),
				),
				StructureStage("Last Stage", StructureStageTaskRef("somepipeline-last-stage-1"),
					StructureStagePrevious("Parent Stage")),
			),
		},
		{
			name: "custom_workspaces",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("stage1",
					StageStep(StepCmd("ls")),
				),
				PipelineStage("stage2",
					StageOptions(
						StageOptionsWorkspace(customWorkspace),
					),
					StageStep(StepCmd("ls")),
				),
				PipelineStage("stage3",
					StageOptions(
						StageOptionsWorkspace(defaultWorkspace),
					),
					StageStep(StepCmd("ls")),
				),
				PipelineStage("stage4",
					StageOptions(
						StageOptionsWorkspace(customWorkspace),
					),
					StageStep(StepCmd("ls")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("stage1", "somepipeline-stage1-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("stage2", "somepipeline-stage2-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage1")),
				tb.PipelineTask("stage3", "somepipeline-stage3-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("stage1")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage2")),
				tb.PipelineTask("stage4", "somepipeline-stage4-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("stage2")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage3")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-stage1-1", "jx", TaskStageLabel("stage1"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage2-1", "jx", TaskStageLabel("stage2"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage3-1", "jx", TaskStageLabel("stage3"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage4-1", "jx", TaskStageLabel("stage4"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("stage1", StructureStageTaskRef("somepipeline-stage1-1")),
				StructureStage("stage2", StructureStageTaskRef("somepipeline-stage2-1"), StructureStagePrevious("stage1")),
				StructureStage("stage3", StructureStageTaskRef("somepipeline-stage3-1"), StructureStagePrevious("stage2")),
				StructureStage("stage4", StructureStageTaskRef("somepipeline-stage4-1"), StructureStagePrevious("stage3")),
			),
		},
		{
			name: "inherited_custom_workspaces",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("stage1",
					StageStep(StepCmd("ls")),
				),
				PipelineStage("stage2",
					StageOptions(
						StageOptionsWorkspace(customWorkspace),
					),
					StageSequential("stage3",
						StageStep(StepCmd("ls")),
					),
					StageSequential("stage4",
						StageOptions(
							StageOptionsWorkspace(defaultWorkspace),
						),
						StageStep(StepCmd("ls")),
					),
					StageSequential("stage5",
						StageStep(StepCmd("ls")),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("stage1", "somepipeline-stage1-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("stage3", "somepipeline-stage3-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage1")),
				tb.PipelineTask("stage4", "somepipeline-stage4-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("stage1")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage3")),
				tb.PipelineTask("stage5", "somepipeline-stage5-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("stage3")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("stage4")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-stage1-1", "jx", TaskStageLabel("stage1"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage3-1", "jx", TaskStageLabel("stage3"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage4-1", "jx", TaskStageLabel("stage4"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-stage5-1", "jx", TaskStageLabel("stage5"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("stage1", StructureStageTaskRef("somepipeline-stage1-1")),
				StructureStage("stage2",
					StructureStagePrevious("stage1"),
					StructureStageStages("stage3", "stage4", "stage5"),
				),
				StructureStage("stage3", StructureStageTaskRef("somepipeline-stage3-1"),
					StructureStageDepth(1),
					StructureStageParent("stage2")),
				StructureStage("stage4", StructureStageTaskRef("somepipeline-stage4-1"),
					StructureStagePrevious("stage3"),
					StructureStageDepth(1),
					StructureStageParent("stage2")),
				StructureStage("stage5", StructureStageTaskRef("somepipeline-stage5-1"),
					StructureStagePrevious("stage4"),
					StructureStageDepth(1),
					StructureStageParent("stage2")),
			),
		},
		{
			name: "environment_at_top_and_in_stage",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineEnvVar("SOME_VAR", "A value for the env var"),
				PipelineStage("A stage with environment",
					StageEnvVar("SOME_OTHER_VAR", "A value for the other env var"),
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("${SOME_OTHER_VAR}")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-stage-with-environment", "somepipeline-a-stage-with-environment-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-stage-with-environment-1", "jx",
					TaskStageLabel("A stage with environment"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${SOME_OTHER_VAR}"), workingDir("/workspace/workspace"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"), tb.EnvVar("SOME_VAR", "A value for the env var")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A stage with environment", StructureStageTaskRef("somepipeline-a-stage-with-environment-1")),
			),
		},
		{
			name: "syntactic_sugar_step_and_a_command",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
					StageStep(StepStep("some-step"),
						StepOptions(map[string]string{"firstParam": "some value", "secondParam": "some other value"})),
				),
			),
			expectedErrorMsg: "syntactic sugar steps not yet supported",
		},
		{
			name: "post",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
					StagePost(syntax.PostConditionSuccess,
						PostAction("mail", map[string]string{
							"to":      "foo@bar.com",
							"subject": "Yay, it passed",
						})),
					StagePost(syntax.PostConditionFailure,
						PostAction("slack", map[string]string{
							"whatever": "the",
							"slack":    "config",
							"actually": "is. =)",
						})),
					StagePost(syntax.PostConditionAlways,
						PostAction("junit", map[string]string{
							"pattern": "target/surefire-reports/**/*.xml",
						}),
					),
				),
			),
			expectedErrorMsg: "post on stages not yet supported",
		},
		{
			name: "top_level_and_stage_options",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineOptions(
					PipelineOptionsTimeout(50, "minutes"),
					PipelineOptionsRetry(3),
				),
				PipelineStage("A Working Stage",
					StageOptions(
						StageOptionsTimeout(5, "seconds"),
						StageOptionsRetry(4),
						StageOptionsStash("Some Files", "somedir/**/*"),
						StageOptionsUnstash("Earlier Files", "some/sub/dir"),
					),
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
				),
			),
			expectedErrorMsg: "Retry at top level not yet supported",
		},
		{
			name: "stage_and_step_agent",
			expected: ParsedPipeline(
				PipelineStage("A Working Stage",
					StageAgent("some-image"),
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
						StepAgent("some-other-image"),
					),
					StageStep(StepCmd("echo"), StepArg("goodbye")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-other-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
						tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo goodbye"), workingDir("/workspace/workspace")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "mangled_task_names",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage(". -a- .",
					StageStep(StepCmd("ls")),
				),
				PipelineStage("Wööh!!!! - This is cool.",
					StageStep(StepCmd("ls")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask(".--a--.", "somepipeline-a-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("wööh!!!!---this-is-cool.", "somepipeline-wh-this-is-cool-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From(".--a--.")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter(".--a--.")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-1", "jx", TaskStageLabel(". -a- ."), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("workspace"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
				)),
				tb.Task("somepipeline-wh-this-is-cool-1", "jx", TaskStageLabel("Wööh!!!! - This is cool."),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/workspace")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage(". -a- .", StructureStageTaskRef("somepipeline-a-1")),
				StructureStage("Wööh!!!! - This is cool.", StructureStageTaskRef("somepipeline-wh-this-is-cool-1"), StructureStagePrevious(". -a- .")),
			),
		},
		{
			name: "stage_timeout",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageOptions(
						StageOptionsTimeout(50, "minutes"),
					),
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
				),
			),
			/* TODO: Stop erroring out once we figure out how to handle task timeouts again
															pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
																tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
																	tb.PipelineTaskInputResource("workspace", "somepipeline"),
																	tb.PipelineTaskInputResource("temp-ordering-resource", "temp-ordering-resource"),
																	tb.PipelineTaskOutputResource("workspace", "somepipeline")),
									tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
																					tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
																	tb.TaskSpec(
												tb.TaskTimeout(50*time.Minute),
																	tb.TaskInputs(
																		tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
																			tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
																	tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
																)),
															},*/
			expectedErrorMsg: "Timeout on stage not yet supported",
		},
		{
			name: "top_level_timeout",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineOptions(
					PipelineOptionsTimeout(50, "minutes"),
				),
				PipelineStage("A Working Stage",
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "loop_step",
			expected: ParsedPipeline(
				PipelineEnvVar("LANGUAGE", "rust"),
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageEnvVar("DISTRO", "gentoo"),
					StageStep(
						StepLoop("LANGUAGE", []string{"maven", "gradle", "nodejs"},
							LoopStep(StepCmd("echo"), StepArg("hello"), StepArg("${LANGUAGE}")),
							LoopStep(StepLoop("DISTRO", []string{"fedora", "ubuntu", "debian"},
								LoopStep(StepCmd("echo"),
									StepArg("running"), StepArg("${LANGUAGE}"),
									StepArg("on"), StepArg("${DISTRO}")),
							)),
						),
					),
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("after")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step4", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step5", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step6", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step7", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step8", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step9", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step10", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step11", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step12", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step13", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step14", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello after"), workingDir("/workspace/workspace"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "rust")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "loop_with_syntactic_sugar_step",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(
						StepLoop("LANGUAGE", []string{"maven", "gradle", "nodejs"},
							LoopStep(StepCmd("echo"), StepArg("hello"), StepArg("${LANGUAGE}")),
							LoopStep(StepStep("some-step"),
								StepOptions(map[string]string{"firstParam": "some value", "secondParam": "some other value"})),
						),
					),
				),
			),
			expectedErrorMsg: "syntactic sugar steps not yet supported",
		},
		{
			name: "top_level_container_options",
			expected: ParsedPipeline(
				PipelineOptions(
					PipelineContainerOptions(
						ContainerResourceLimits("0.2", "128Mi"),
						ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace"),
							ContainerResourceLimits("0.2", "128Mi"),
							ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "stage_overrides_top_level_container_options",
			expected: ParsedPipeline(
				PipelineOptions(
					PipelineContainerOptions(
						ContainerResourceLimits("0.2", "128Mi"),
						ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageOptions(
						StageContainerOptions(
							ContainerResourceLimits("0.4", "256Mi"),
							ContainerResourceRequests("0.2", "128Mi"),
						),
					),
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace"),
							ContainerResourceLimits("0.4", "256Mi"),
							ContainerResourceRequests("0.2", "128Mi"),
						),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "merge_container_options",
			expected: ParsedPipeline(
				PipelineOptions(
					PipelineContainerOptions(
						ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageOptions(
						StageContainerOptions(
							ContainerResourceLimits("0.4", "256Mi"),
						),
					),
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace"),
							ContainerResourceLimits("0.4", "256Mi"),
							ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "stage_level_container_options",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageOptions(
						StageContainerOptions(
							ContainerResourceLimits("0.2", "128Mi"),
							ContainerResourceRequests("0.1", "64Mi"),
						),
					),
					StageStep(
						StepCmd("echo"),
						StepArg("hello"), StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace"),
							ContainerResourceLimits("0.2", "128Mi"),
							ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "container_options_env_merge",
			expected: ParsedPipeline(
				PipelineAgent("some-image"),
				PipelineOptions(
					PipelineContainerOptions(
						tb.EnvVar("SOME_VAR", "A value for the env var"),
						tb.EnvVar("OVERRIDE_ENV", "Original value"),
						tb.EnvVar("OVERRIDE_STAGE_ENV", "Original value"),
					),
				),
				PipelineEnvVar("SOME_OTHER_VAR", "A value for the other env var"),
				PipelineEnvVar("OVERRIDE_ENV", "New value"),
				PipelineStage("A Working Stage",
					StageOptions(
						StageContainerOptions(
							tb.EnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "Original value"),
						),
					),
					StageStep(StepCmd("echo"), StepArg("hello"), StepArg("world")),
					StageEnvVar("OVERRIDE_STAGE_ENV", "New value"),
					StageEnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "New value"),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx",
					TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("workspace"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/workspace"),
							tb.EnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("OVERRIDE_ENV", "New value"),
							tb.EnvVar("OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"),
							tb.EnvVar("SOME_VAR", "A value for the env var"),
						),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConfig, fn, err := config.LoadProjectConfig(filepath.Join("test_data", tt.name))
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

			if d, _ := kmp.SafeDiff(tt.expected, parsed); d != "" && tt.expected != nil {
				t.Errorf("Parsed ParsedPipeline did not match expected: %s", d)
			}

			validateErr := parsed.Validate(ctx)
			if validateErr != nil && tt.validationErrorMsg == "" {
				t.Errorf("Validation failed: %s", validateErr)
			}

			if validateErr != nil && tt.validationErrorMsg != "" {
				if tt.validationErrorMsg != validateErr.Details {
					t.Errorf("Validation Error failed: '%s', '%s'", validateErr.Details, tt.validationErrorMsg)
				}
			}

			pipeline, tasks, structure, err := parsed.GenerateCRDs("somepipeline", "1", "jx", nil, nil, "workspace")

			if err != nil {
				if tt.expectedErrorMsg != "" {
					if d := cmp.Diff(tt.expectedErrorMsg, err.Error()); d != "" {
						t.Fatalf("CRD generation error did not meet expectation: %s", d)
					}
				} else {
					t.Fatalf("Error generating CRDs: %s", err)
				}
			}

			if tt.expectedErrorMsg == "" && tt.pipeline != nil {
				pipeline.TypeMeta = metav1.TypeMeta{}
				if d := cmp.Diff(tt.pipeline, pipeline); d != "" {
					t.Errorf("Generated Pipeline did not match expected: %s", d)
				}

				if err := pipeline.Spec.Validate(ctx); err != nil {
					t.Errorf("PipelineSpec.Validate(ctx) = %v", err)
				}

				for _, task := range tasks {
					task.TypeMeta = metav1.TypeMeta{}
				}
				if d, _ := kmp.SafeDiff(tt.tasks, tasks); d != "" {
					t.Errorf("Generated Tasks did not match expected: %s", d)
				}

				for _, task := range tasks {
					if err := task.Spec.Validate(ctx); err != nil {
						t.Errorf("TaskSpec.Validate(ctx) = %v", err)
					}
				}

				if tt.structure != nil {
					if d := cmp.Diff(tt.structure, structure); d != "" {
						t.Errorf("Generated PipelineStructure did not match expected: %s", d)
					}
				}
			}
		})
	}
}

func TestFailedValidation(t *testing.T) {
	ctx := context.Background()
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
			name:          "step_without_command_step_or_loop",
			expectedError: apis.ErrMissingOneOf("command", "step", "loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name:          "step_with_both_command_and_step",
			expectedError: apis.ErrMultipleOneOf("command", "step", "loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name:          "step_with_both_command_and_loop",
			expectedError: apis.ErrMultipleOneOf("command", "step", "loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_command_and_options",
			expectedError: (&apis.FieldError{
				Message: "Cannot set options for a command or a loop",
				Paths:   []string{"options"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_step_and_arguments",
			expectedError: (&apis.FieldError{
				Message: "Cannot set command-line arguments for a step or a loop",
				Paths:   []string{"args"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_loop_and_options",
			expectedError: (&apis.FieldError{
				Message: "Cannot set options for a command or a loop",
				Paths:   []string{"options"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "step_with_loop_and_arguments",
			expectedError: (&apis.FieldError{
				Message: "Cannot set command-line arguments for a step or a loop",
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
		{
			name: "stage_name_duplicates",
			expectedError: &apis.FieldError{
				Message: "Stage names must be unique",
				Details: "The following stage names are used more than once: 'A Working Stage'",
			},
		},
		{
			name: "stage_name_duplicates_deeply_nested",
			expectedError: &apis.FieldError{
				Message: "Stage names must be unique",
				Details: "The following stage names are used more than once: 'Stage With Stages'",
			},
		},
		{
			name: "stage_name_duplicates_nested",
			expectedError: &apis.FieldError{
				Message: "Stage names must be unique",
				Details: "The following stage names are used more than once: 'Stage With Stages'",
			},
		},
		{
			name: "stage_name_duplicates_sequential",
			expectedError: &apis.FieldError{
				Message: "Stage names must be unique",
				Details: "The following stage names are used more than once: 'A Working title 2', 'A Working title'",
			},
		},
		{
			name: "stage_name_duplicates_unique_in_scope",
			expectedError: &apis.FieldError{
				Message: "Stage names must be unique",
				Details: "The following stage names are used more than once: 'A Working title 1', 'A Working title 2'",
			},
		},
		{
			name:          "loop_without_variable",
			expectedError: apis.ErrMissingField("variable").ViaField("loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name:          "loop_without_steps",
			expectedError: apis.ErrMissingField("steps").ViaField("loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name:          "loop_without_values",
			expectedError: apis.ErrMissingField("values").ViaField("loop").ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "top_level_container_options_with_command",
			expectedError: (&apis.FieldError{
				Message: "Command cannot be specified in containerOptions",
				Paths:   []string{"command"},
			}).ViaField("containerOptions").ViaField("options"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConfig, fn, err := config.LoadProjectConfig(filepath.Join("test_data", "validation_failures", tt.name))
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

			err = parsed.Validate(ctx)

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

// Command sets the command to the Container (step in this case).
func workingDir(dir string) tb.ContainerOp {
	return func(container *corev1.Container) {
		container.WorkingDir = dir
	}
}

type PipelineStructureOp func(structure *v1.PipelineStructure)
type PipelineStructureStageOp func(stage *v1.PipelineStructureStage)

func PipelineStructure(name string, ops ...PipelineStructureOp) *v1.PipelineStructure {
	s := &v1.PipelineStructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	for _, op := range ops {
		op(s)
	}

	return s
}

func StructurePipelineRunRef(name string) PipelineStructureOp {
	return func(structure *v1.PipelineStructure) {
		structure.PipelineRunRef = &name
	}
}

func StructureStage(name string, ops ...PipelineStructureStageOp) PipelineStructureOp {
	return func(structure *v1.PipelineStructure) {
		stage := v1.PipelineStructureStage{Name: name}

		for _, op := range ops {
			op(&stage)
		}

		structure.Stages = append(structure.Stages, stage)
	}
}

func StructureStageTaskRef(name string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.TaskRef = &name
	}
}

func StructureStageTaskRunRef(name string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.TaskRunRef = &name
	}
}

func StructureStageDepth(depth int8) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Depth = depth
	}
}

func StructureStageParent(parent string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Parent = &parent
	}
}

func StructureStagePrevious(previous string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Previous = &previous
	}
}

func StructureStageNext(Next string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Next = &Next
	}
}

func StructureStageStages(stages ...string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Stages = append(stage.Stages, stages...)
	}
}

func StructureStageParallel(stages ...string) PipelineStructureStageOp {
	return func(stage *v1.PipelineStructureStage) {
		stage.Parallel = append(stage.Parallel, stages...)
	}
}

type PipelineOp func(*syntax.ParsedPipeline)
type PipelineOptionsOp func(*syntax.RootOptions)
type PipelinePostOp func(*syntax.Post)
type StageOp func(*syntax.Stage)
type StageOptionsOp func(*syntax.StageOptions)
type StepOp func(*syntax.Step)
type LoopOp func(*syntax.Loop)

func ParsedPipeline(ops ...PipelineOp) *syntax.ParsedPipeline {
	s := &syntax.ParsedPipeline{}

	for _, op := range ops {
		op(s)
	}

	return s
}

func PipelineAgent(image string) PipelineOp {
	return func(parsed *syntax.ParsedPipeline) {
		parsed.Agent = syntax.Agent{
			Image: image,
		}
	}
}

func PipelineOptions(ops ...PipelineOptionsOp) PipelineOp {
	return func(parsed *syntax.ParsedPipeline) {
		parsed.Options = syntax.RootOptions{}

		for _, op := range ops {
			op(&parsed.Options)
		}
	}
}

func PipelineContainerOptions(ops ...tb.ContainerOp) PipelineOptionsOp {
	return func(options *syntax.RootOptions) {
		options.ContainerOptions = &corev1.Container{}

		for _, op := range ops {
			op(options.ContainerOptions)
		}
	}
}

func StageContainerOptions(ops ...tb.ContainerOp) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.ContainerOptions = &corev1.Container{}

		for _, op := range ops {
			op(options.ContainerOptions)
		}
	}
}

func ContainerResourceLimits(cpus, memory string) tb.ContainerOp {
	return func(container *corev1.Container) {
		cpuQuantity, _ := resource.ParseQuantity(cpus)
		memoryQuantity, _ := resource.ParseQuantity(memory)
		container.Resources.Limits = corev1.ResourceList{
			"cpu":    cpuQuantity,
			"memory": memoryQuantity,
		}
	}
}

func ContainerResourceRequests(cpus, memory string) tb.ContainerOp {
	return func(container *corev1.Container) {
		cpuQuantity, _ := resource.ParseQuantity(cpus)
		memoryQuantity, _ := resource.ParseQuantity(memory)
		container.Resources.Requests = corev1.ResourceList{
			"cpu":    cpuQuantity,
			"memory": memoryQuantity,
		}
	}
}

func PipelineOptionsTimeout(time int64, unit syntax.TimeoutUnit) PipelineOptionsOp {
	return func(options *syntax.RootOptions) {
		options.Timeout = syntax.Timeout{
			Time: time,
			Unit: unit,
		}
	}
}

func PipelineOptionsRetry(count int8) PipelineOptionsOp {
	return func(options *syntax.RootOptions) {
		options.Retry = count
	}
}

// PipelineEnvVar add an environment variable, with specified name and value, to the pipeline.
func PipelineEnvVar(name, value string) PipelineOp {
	return func(parsed *syntax.ParsedPipeline) {
		parsed.Environment = append(parsed.Environment, syntax.EnvVar{
			Name:  name,
			Value: value,
		})
	}
}

func PipelinePost(condition syntax.PostCondition, ops ...PipelinePostOp) PipelineOp {
	return func(parsed *syntax.ParsedPipeline) {
		post := syntax.Post{
			Condition: condition,
		}

		for _, op := range ops {
			op(&post)
		}

		parsed.Post = append(parsed.Post, post)
	}
}

func PipelineStage(name string, ops ...StageOp) PipelineOp {
	return func(parsed *syntax.ParsedPipeline) {
		s := syntax.Stage{
			Name: name,
		}

		for _, op := range ops {
			op(&s)
		}
		parsed.Stages = append(parsed.Stages, s)
	}
}

func PostAction(name string, options map[string]string) PipelinePostOp {
	return func(post *syntax.Post) {
		post.Actions = append(post.Actions, syntax.PostAction{
			Name:    name,
			Options: options,
		})
	}
}

func StageAgent(image string) StageOp {
	return func(stage *syntax.Stage) {
		stage.Agent = syntax.Agent{
			Image: image,
		}
	}
}

func StageOptions(ops ...StageOptionsOp) StageOp {
	return func(stage *syntax.Stage) {
		stage.Options = syntax.StageOptions{}

		for _, op := range ops {
			op(&stage.Options)
		}
	}
}

func StageOptionsTimeout(time int64, unit syntax.TimeoutUnit) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.Timeout = syntax.Timeout{
			Time: time,
			Unit: unit,
		}
	}
}

func StageOptionsRetry(count int8) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.Retry = count
	}
}

func StageOptionsWorkspace(ws string) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.Workspace = &ws
	}
}

func StageOptionsStash(name, files string) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.Stash = syntax.Stash{
			Name:  name,
			Files: files,
		}
	}
}

func StageOptionsUnstash(name, dir string) StageOptionsOp {
	return func(options *syntax.StageOptions) {
		options.Unstash = syntax.Unstash{
			Name: name,
		}
		if dir != "" {
			options.Unstash.Dir = dir
		}
	}
}

// AgentEnvVar add an environment variable, with specified name and value, to the stage.
func StageEnvVar(name, value string) StageOp {
	return func(stage *syntax.Stage) {
		stage.Environment = append(stage.Environment, syntax.EnvVar{
			Name:  name,
			Value: value,
		})
	}
}

func StagePost(condition syntax.PostCondition, ops ...PipelinePostOp) StageOp {
	return func(stage *syntax.Stage) {
		post := syntax.Post{
			Condition: condition,
		}

		for _, op := range ops {
			op(&post)
		}

		stage.Post = append(stage.Post, post)
	}
}

func StepAgent(image string) StepOp {
	return func(step *syntax.Step) {
		step.Agent = syntax.Agent{
			Image: image,
		}
	}
}

func StepCmd(cmd string) StepOp {
	return func(step *syntax.Step) {
		step.Command = cmd
	}
}

func StepArg(arg string) StepOp {
	return func(step *syntax.Step) {
		step.Arguments = append(step.Arguments, arg)
	}
}

func StepStep(s string) StepOp {
	return func(step *syntax.Step) {
		step.Step = s
	}
}

func StepOptions(options map[string]string) StepOp {
	return func(step *syntax.Step) {
		step.Options = options
	}
}

func StepDir(dir string) StepOp {
	return func(step *syntax.Step) {
		step.Dir = dir
	}
}

func StepLoop(variable string, values []string, ops ...LoopOp) StepOp {
	return func(step *syntax.Step) {
		loop := syntax.Loop{
			Variable: variable,
			Values:   values,
		}

		for _, op := range ops {
			op(&loop)
		}

		step.Loop = loop
	}
}

func LoopStep(ops ...StepOp) LoopOp {
	return func(loop *syntax.Loop) {
		step := syntax.Step{}

		for _, op := range ops {
			op(&step)
		}

		loop.Steps = append(loop.Steps, step)
	}
}

func StageStep(ops ...StepOp) StageOp {
	return func(stage *syntax.Stage) {
		step := syntax.Step{}

		for _, op := range ops {
			op(&step)
		}

		stage.Steps = append(stage.Steps, step)
	}
}

func StageParallel(name string, ops ...StageOp) StageOp {
	return func(stage *syntax.Stage) {
		n := syntax.Stage{Name: name}

		for _, op := range ops {
			op(&n)
		}

		stage.Parallel = append(stage.Parallel, n)
	}
}

func StageSequential(name string, ops ...StageOp) StageOp {
	return func(stage *syntax.Stage) {
		n := syntax.Stage{Name: name}

		for _, op := range ops {
			op(&n)
		}

		stage.Stages = append(stage.Stages, n)
	}
}

func TaskStageLabel(value string) tb.TaskOp {
	return func(t *tektonv1alpha1.Task) {
		if t.ObjectMeta.Labels == nil {
			t.ObjectMeta.Labels = map[string]string{}
		}
		t.ObjectMeta.Labels[syntax.LabelStageName] = syntax.MangleToRfc1035Label(value, "")
	}
}

func TestParsedPipelineHelpers(t *testing.T) {
	input := ParsedPipeline(
		PipelineAgent("some-image"),
		PipelineOptions(
			PipelineOptionsRetry(5),
			PipelineOptionsTimeout(30, syntax.TimeoutUnitSeconds),
		),
		PipelineEnvVar("ANIMAL", "MONKEY"),
		PipelineEnvVar("FRUIT", "BANANA"),
		PipelinePost(syntax.PostConditionSuccess,
			PostAction("mail", map[string]string{
				"to":      "foo@bar.com",
				"subject": "Yay, it passed",
			})),
		PipelinePost(syntax.PostConditionFailure,
			PostAction("slack", map[string]string{
				"whatever": "the",
				"slack":    "config",
				"actually": "is. =)",
			})),
		PipelineStage("A Working Stage",
			StageOptions(
				StageOptionsWorkspace(customWorkspace),
				StageOptionsStash("some-name", "**/*"),
				StageOptionsUnstash("some-name", ""),
				StageOptionsTimeout(15, syntax.TimeoutUnitMinutes),
				StageOptionsRetry(2),
			),
			StageStep(
				StepCmd("echo"),
				StepArg("hello"),
				StepArg("world"),
			),
		),
		PipelineStage("Parent Stage",
			StageParallel("First Nested Stage",
				StageAgent("some-other-image"),
				StageStep(
					StepCmd("echo"),
					StepArg("hello"),
					StepArg("world"),
					StepAgent("some-other-image"),
				),
				StageEnvVar("STAGE_VAR_ONE", "some value"),
				StageEnvVar("STAGE_VAR_TWO", "some other value"),
				StagePost(syntax.PostConditionAlways,
					PostAction("junit", map[string]string{
						"pattern": "target/surefire-reports/**/*.xml",
					}),
				),
			),
			StageParallel("Nested In Parallel",
				StageSequential("Another stage",
					StageStep(
						StepLoop("SOME_VAR", []string{"a", "b", "c"},
							LoopStep(
								StepCmd("echo"),
								StepArg("SOME_VAR is ${SOME_VAR}"),
							),
						),
					),
				),
				StageSequential("Some other stage",
					StageStep(
						StepCmd("echo"),
						StepArg("otherwise"),
						StepDir(customWorkspace),
					),
					StageStep(
						StepStep("some-step"),
						StepOptions(map[string]string{"first": "arg", "second": "arg"}),
					),
				),
			),
		),
	)

	expected := &syntax.ParsedPipeline{
		Agent: syntax.Agent{
			Image: "some-image",
		},
		Options: syntax.RootOptions{
			Retry: 5,
			Timeout: syntax.Timeout{
				Time: 30,
				Unit: syntax.TimeoutUnitSeconds,
			},
		},
		Environment: []syntax.EnvVar{
			{
				Name:  "ANIMAL",
				Value: "MONKEY",
			},
			{
				Name:  "FRUIT",
				Value: "BANANA",
			},
		},
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
		},
		Stages: []syntax.Stage{
			{
				Name: "A Working Stage",
				Options: syntax.StageOptions{
					Workspace: &customWorkspace,
					Stash: syntax.Stash{
						Name:  "some-name",
						Files: "**/*",
					},
					Unstash: syntax.Unstash{
						Name: "some-name",
					},
					RootOptions: syntax.RootOptions{
						Timeout: syntax.Timeout{
							Time: 15,
							Unit: syntax.TimeoutUnitMinutes,
						},
						Retry: 2,
					},
				},
				Steps: []syntax.Step{{
					Command:   "echo",
					Arguments: []string{"hello", "world"},
				}},
			},
			{
				Name: "Parent Stage",
				Parallel: []syntax.Stage{
					{
						Name: "First Nested Stage",
						Agent: syntax.Agent{
							Image: "some-other-image",
						},
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"hello", "world"},
							Agent: syntax.Agent{
								Image: "some-other-image",
							},
						}},
						Environment: []syntax.EnvVar{
							{
								Name:  "STAGE_VAR_ONE",
								Value: "some value",
							},
							{
								Name:  "STAGE_VAR_TWO",
								Value: "some other value",
							},
						},
						Post: []syntax.Post{{
							Condition: "always",
							Actions: []syntax.PostAction{{
								Name: "junit",
								Options: map[string]string{
									"pattern": "target/surefire-reports/**/*.xml",
								},
							}},
						}},
					},
					{
						Name: "Nested In Parallel",
						Stages: []syntax.Stage{
							{
								Name: "Another stage",
								Steps: []syntax.Step{{
									Loop: syntax.Loop{
										Variable: "SOME_VAR",
										Values:   []string{"a", "b", "c"},
										Steps: []syntax.Step{{
											Command:   "echo",
											Arguments: []string{"SOME_VAR is ${SOME_VAR}"},
										}},
									},
								}},
							},
							{
								Name: "Some other stage",
								Steps: []syntax.Step{
									{
										Command:   "echo",
										Arguments: []string{"otherwise"},
										Dir:       customWorkspace,
									},
									{
										Step:    "some-step",
										Options: map[string]string{"first": "arg", "second": "arg"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if d := cmp.Diff(expected, input); d != "" {
		t.Fatalf("ParsedPipeline diff -want, +got: %v", d)
	}
}
