package syntax_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/tekton/syntax/syntax_helpers_test"
	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/kmp"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
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
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
						syntax_helpers_test.StepName("A Step With Spaces And Such"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
						tb.Step("a-step-with-spaces-and-such", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "multiple_stages",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
				),
				syntax_helpers_test.PipelineStage("Another stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("again"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.RunAfter("a-working-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-another-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
				syntax_helpers_test.StructureStage("Another stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-another-stage-1"),
					syntax_helpers_test.StructureStagePrevious("A Working Stage")),
			),
		},
		{
			name: "nested_stages",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("Parent Stage",
					syntax_helpers_test.StageSequential("A Working Stage",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"))),
					syntax_helpers_test.StageSequential("Another stage",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("again"))),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.RunAfter("a-working-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit, tb.ResourceTargetPath("source"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("Parent Stage",
					syntax_helpers_test.StructureStageStages("A Working Stage", "Another stage")),
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("Parent Stage")),
				syntax_helpers_test.StructureStage("Another stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-another-stage-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("Parent Stage"),
					syntax_helpers_test.StructureStagePrevious("A Working Stage")),
			),
		},
		{
			name: "parallel_stages",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("First Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("first"))),
				syntax_helpers_test.PipelineStage("Parent Stage",
					syntax_helpers_test.StageParallel("A Working Stage",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"))),
					syntax_helpers_test.StageParallel("Another stage",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("again"))),
				),
				syntax_helpers_test.PipelineStage("Last Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("last"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline", tb.From("first-stage")),
					tb.RunAfter("a-working-stage", "another-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-first-stage-1", "jx", syntax_helpers_test.TaskStageLabel("First Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo first"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-last-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Last Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo last"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("First Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-first-stage-1")),
				syntax_helpers_test.StructureStage("Parent Stage",
					syntax_helpers_test.StructureStageParallel("A Working Stage", "Another stage"),
					syntax_helpers_test.StructureStagePrevious("First Stage"),
				),
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("Parent Stage"),
				),
				syntax_helpers_test.StructureStage("Another stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-another-stage-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("Parent Stage"),
				),
				syntax_helpers_test.StructureStage("Last Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-last-stage-1"),
					syntax_helpers_test.StructureStagePrevious("Parent Stage")),
			),
		},
		{
			name: "parallel_and_nested_stages",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("First Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("first"))),
				syntax_helpers_test.PipelineStage("Parent Stage",
					syntax_helpers_test.StageParallel("A Working Stage",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"))),
					syntax_helpers_test.StageParallel("Nested In Parallel",
						syntax_helpers_test.StageSequential("Another stage",
							syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("again"))),
						syntax_helpers_test.StageSequential("Some other stage",
							syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("otherwise"))),
					),
				),
				syntax_helpers_test.PipelineStage("Last Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("last"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("first-stage", "somepipeline-first-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.PipelineTaskOutputResource("workspace", "somepipeline"),
					tb.RunAfter("first-stage")),
				tb.PipelineTask("some-other-stage", "somepipeline-some-other-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("another-stage")),
					tb.RunAfter("another-stage")),
				tb.PipelineTask("last-stage", "somepipeline-last-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("first-stage")),
					tb.RunAfter("a-working-stage", "some-other-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-first-stage-1", "jx", syntax_helpers_test.TaskStageLabel("First Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo first"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
					)),
				tb.Task("somepipeline-another-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-some-other-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Some other stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo otherwise"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-last-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Last Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo last"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("First Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-first-stage-1")),
				syntax_helpers_test.StructureStage("Parent Stage",
					syntax_helpers_test.StructureStageParallel("A Working Stage", "Nested In Parallel"),
					syntax_helpers_test.StructureStagePrevious("First Stage"),
				),
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("Parent Stage"),
				),
				syntax_helpers_test.StructureStage("Nested In Parallel",
					syntax_helpers_test.StructureStageParent("Parent Stage"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageStages("Another stage", "Some other stage"),
				),
				syntax_helpers_test.StructureStage("Another stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-another-stage-1"),
					syntax_helpers_test.StructureStageDepth(2),
					syntax_helpers_test.StructureStageParent("Nested In Parallel"),
				),
				syntax_helpers_test.StructureStage("Some other stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-some-other-stage-1"),
					syntax_helpers_test.StructureStageDepth(2),
					syntax_helpers_test.StructureStageParent("Nested In Parallel"),
					syntax_helpers_test.StructureStagePrevious("Another stage"),
				),
				syntax_helpers_test.StructureStage("Last Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-last-stage-1"),
					syntax_helpers_test.StructureStagePrevious("Parent Stage")),
			),
		},
		{
			name: "custom_workspaces",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("stage1",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
				syntax_helpers_test.PipelineStage("stage2",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsWorkspace(customWorkspace),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
				syntax_helpers_test.PipelineStage("stage3",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsWorkspace(defaultWorkspace),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
				syntax_helpers_test.PipelineStage("stage4",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsWorkspace(customWorkspace),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
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
					tb.RunAfter("stage3")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-stage1-1", "jx", syntax_helpers_test.TaskStageLabel("stage1"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage2-1", "jx", syntax_helpers_test.TaskStageLabel("stage2"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage3-1", "jx", syntax_helpers_test.TaskStageLabel("stage3"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage4-1", "jx", syntax_helpers_test.TaskStageLabel("stage4"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("stage1", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage1-1")),
				syntax_helpers_test.StructureStage("stage2", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage2-1"), syntax_helpers_test.StructureStagePrevious("stage1")),
				syntax_helpers_test.StructureStage("stage3", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage3-1"), syntax_helpers_test.StructureStagePrevious("stage2")),
				syntax_helpers_test.StructureStage("stage4", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage4-1"), syntax_helpers_test.StructureStagePrevious("stage3")),
			),
		},
		{
			name: "inherited_custom_workspaces",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("stage1",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
				syntax_helpers_test.PipelineStage("stage2",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsWorkspace(customWorkspace),
					),
					syntax_helpers_test.StageSequential("stage3",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
					),
					syntax_helpers_test.StageSequential("stage4",
						syntax_helpers_test.StageOptions(
							syntax_helpers_test.StageOptionsWorkspace(defaultWorkspace),
						),
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
					),
					syntax_helpers_test.StageSequential("stage5",
						syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
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
					tb.RunAfter("stage4")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-stage1-1", "jx", syntax_helpers_test.TaskStageLabel("stage1"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage3-1", "jx", syntax_helpers_test.TaskStageLabel("stage3"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage4-1", "jx", syntax_helpers_test.TaskStageLabel("stage4"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-stage5-1", "jx", syntax_helpers_test.TaskStageLabel("stage5"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("stage1", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage1-1")),
				syntax_helpers_test.StructureStage("stage2",
					syntax_helpers_test.StructureStagePrevious("stage1"),
					syntax_helpers_test.StructureStageStages("stage3", "stage4", "stage5"),
				),
				syntax_helpers_test.StructureStage("stage3", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage3-1"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("stage2")),
				syntax_helpers_test.StructureStage("stage4", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage4-1"),
					syntax_helpers_test.StructureStagePrevious("stage3"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("stage2")),
				syntax_helpers_test.StructureStage("stage5", syntax_helpers_test.StructureStageTaskRef("somepipeline-stage5-1"),
					syntax_helpers_test.StructureStagePrevious("stage4"),
					syntax_helpers_test.StructureStageDepth(1),
					syntax_helpers_test.StructureStageParent("stage2")),
			),
		},
		{
			name: "environment_at_top_and_in_stage",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineEnvVar("SOME_VAR", "A value for the env var"),
				syntax_helpers_test.PipelineStage("A stage with environment",
					syntax_helpers_test.StageEnvVar("SOME_OTHER_VAR", "A value for the other env var"),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("${SOME_OTHER_VAR}")),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("goodbye"), syntax_helpers_test.StepArg("${SOME_VAR} and ${ANOTHER_VAR}"),
						syntax_helpers_test.StepEnvVar("SOME_VAR", "An overriding value"),
						syntax_helpers_test.StepEnvVar("ANOTHER_VAR", "Yet another variable"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-stage-with-environment", "somepipeline-a-stage-with-environment-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-stage-with-environment-1", "jx",
					syntax_helpers_test.TaskStageLabel("A stage with environment"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"), tb.EnvVar("SOME_VAR", "A value for the env var")),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${SOME_OTHER_VAR}"), workingDir("/workspace/source"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"), tb.EnvVar("SOME_VAR", "A value for the env var")),
						tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo goodbye ${SOME_VAR} and ${ANOTHER_VAR}"), workingDir("/workspace/source"),
							tb.EnvVar("ANOTHER_VAR", "Yet another variable"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"),
							tb.EnvVar("SOME_VAR", "An overriding value"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A stage with environment", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-stage-with-environment-1")),
			),
		},
		{
			name: "syntactic_sugar_step_and_a_command",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepStep("some-step"),
						syntax_helpers_test.StepOptions(map[string]string{"firstParam": "some value", "secondParam": "some other value"})),
				),
			),
			expectedErrorMsg: "syntactic sugar steps not yet supported",
		},
		{
			name: "post",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
					syntax_helpers_test.StagePost(syntax.PostConditionSuccess,
						syntax_helpers_test.PostAction("mail", map[string]string{
							"to":      "foo@bar.com",
							"subject": "Yay, it passed",
						})),
					syntax_helpers_test.StagePost(syntax.PostConditionFailure,
						syntax_helpers_test.PostAction("slack", map[string]string{
							"whatever": "the",
							"slack":    "config",
							"actually": "is. =)",
						})),
					syntax_helpers_test.StagePost(syntax.PostConditionAlways,
						syntax_helpers_test.PostAction("junit", map[string]string{
							"pattern": "target/surefire-reports/**/*.xml",
						}),
					),
				),
			),
			expectedErrorMsg: "post on stages not yet supported",
		},
		{
			name: "top_level_and_stage_options",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineOptionsTimeout(50, "minutes"),
					syntax_helpers_test.PipelineOptionsRetry(3),
				),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsTimeout(5, "seconds"),
						syntax_helpers_test.StageOptionsRetry(4),
						syntax_helpers_test.StageOptionsStash("Some Files", "somedir/**/*"),
						syntax_helpers_test.StageOptionsUnstash("Earlier Files", "some/sub/dir"),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
				),
			),
			expectedErrorMsg: "Retry at top level not yet supported",
		},
		{
			name: "stage_and_step_agent",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageAgent("some-image"),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
						syntax_helpers_test.StepAgent("some-other-image"),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("goodbye")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
						tb.Step("step2", "some-other-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
						tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo goodbye"), workingDir("/workspace/source")),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "mangled_task_names",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage(". -a- .",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
				syntax_helpers_test.PipelineStage("Wööh!!!! - This is cool.",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("ls")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask(".--a--.", "somepipeline-a-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("wööh!!!!---this-is-cool.", "somepipeline-wh-this-is-cool-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From(".--a--.")),
					tb.RunAfter(".--a--.")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-1", "jx", syntax_helpers_test.TaskStageLabel(". -a- ."), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
				)),
				tb.Task("somepipeline-wh-this-is-cool-1", "jx", syntax_helpers_test.TaskStageLabel("Wööh!!!! - This is cool."),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("ls"), workingDir("/workspace/source")),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage(". -a- .", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-1")),
				syntax_helpers_test.StructureStage("Wööh!!!! - This is cool.", syntax_helpers_test.StructureStageTaskRef("somepipeline-wh-this-is-cool-1"), syntax_helpers_test.StructureStagePrevious(". -a- .")),
			),
		},
		{
			name: "stage_timeout",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageOptionsTimeout(50, "minutes"),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
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
																			tb.ResourceTargetPath("source"))),
						tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
																	tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
																)),
															},*/
			expectedErrorMsg: "Timeout on stage not yet supported",
		},
		{
			name: "top_level_timeout",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineOptionsTimeout(50, "minutes"),
				),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source")),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "loop_step",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineEnvVar("LANGUAGE", "rust"),
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageEnvVar("DISTRO", "gentoo"),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepLoop("LANGUAGE", []string{"maven", "gradle", "nodejs"},
							syntax_helpers_test.LoopStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("${LANGUAGE}")),
							syntax_helpers_test.LoopStep(syntax_helpers_test.StepLoop("DISTRO", []string{"fedora", "ubuntu", "debian"},
								syntax_helpers_test.LoopStep(syntax_helpers_test.StepCmd("echo"),
									syntax_helpers_test.StepArg("running"), syntax_helpers_test.StepArg("${LANGUAGE}"),
									syntax_helpers_test.StepArg("on"), syntax_helpers_test.StepArg("${DISTRO}")),
							)),
						),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("after")),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "rust")),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step4", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step5", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("step6", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step7", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step8", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step9", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("step10", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step11", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "fedora"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step12", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "ubuntu"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step13", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo running ${LANGUAGE} on ${DISTRO}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "debian"), tb.EnvVar("LANGUAGE", "nodejs")),
						tb.Step("step14", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello after"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "rust")),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "loop_step_with_name",
			expected: ParsedPipeline(
				PipelineEnvVar("LANGUAGE", "rust"),
				PipelineAgent("some-image"),
				PipelineStage("A Working Stage",
					StageEnvVar("DISTRO", "gentoo"),
					StageStep(
						StepLoop("LANGUAGE", []string{"maven", "gradle", "nodejs"},
							LoopStep(StepName("echo-step"), StepCmd("echo"), StepArg("hello"), StepArg("${LANGUAGE}")),
						),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "rust")),
						tb.Step("echo-step1", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "maven")),
						tb.Step("echo-step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "gradle")),
						tb.Step("echo-step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello ${LANGUAGE}"), workingDir("/workspace/source"),
							tb.EnvVar("DISTRO", "gentoo"), tb.EnvVar("LANGUAGE", "nodejs")),
					)),
			},
			structure: PipelineStructure("somepipeline-1",
				StructureStage("A Working Stage", StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "loop_with_syntactic_sugar_step",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepLoop("LANGUAGE", []string{"maven", "gradle", "nodejs"},
							syntax_helpers_test.LoopStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("${LANGUAGE}")),
							syntax_helpers_test.LoopStep(syntax_helpers_test.StepStep("some-step"),
								syntax_helpers_test.StepOptions(map[string]string{"firstParam": "some value", "secondParam": "some other value"})),
						),
					),
				),
			),
			expectedErrorMsg: "syntactic sugar steps not yet supported",
		},
		{
			name: "top_level_container_options",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineContainerOptions(
						syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
						syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "stage_overrides_top_level_container_options",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineContainerOptions(
						syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
						syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageContainerOptions(
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.2", "128Mi"),
						),
					),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.2", "128Mi"),
						),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.2", "128Mi"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "merge_container_options",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineContainerOptions(
						syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
					),
				),
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageContainerOptions(
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
						),
					),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.4", "256Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "stage_level_container_options",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageContainerOptions(
							syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
					),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world"),
					),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source"),
							syntax_helpers_test.ContainerResourceLimits("0.2", "128Mi"),
							syntax_helpers_test.ContainerResourceRequests("0.1", "64Mi"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "container_options_env_merge",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineOptions(
					syntax_helpers_test.PipelineContainerOptions(
						tb.EnvVar("SOME_VAR", "A value for the env var"),
						tb.EnvVar("OVERRIDE_ENV", "Original value"),
						tb.EnvVar("OVERRIDE_STAGE_ENV", "Original value"),
					),
				),
				syntax_helpers_test.PipelineEnvVar("SOME_OTHER_VAR", "A value for the other env var"),
				syntax_helpers_test.PipelineEnvVar("OVERRIDE_ENV", "New value"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageOptions(
						syntax_helpers_test.StageContainerOptions(
							tb.EnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "Original value"),
						),
					),
					syntax_helpers_test.StageStep(syntax_helpers_test.StepCmd("echo"), syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
					syntax_helpers_test.StageEnvVar("OVERRIDE_STAGE_ENV", "New value"),
					syntax_helpers_test.StageEnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "New value"),
				),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
				),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx",
					syntax_helpers_test.TaskStageLabel("A Working Stage"),
					tb.TaskSpec(
						tb.TaskInputs(
							tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
								tb.ResourceTargetPath("source"))),
						tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source"),
							tb.EnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("SOME_VAR", "A value for the env var"),
							tb.EnvVar("OVERRIDE_ENV", "New value"),
							tb.EnvVar("OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"),
						),
						tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source"),
							tb.EnvVar("ANOTHER_OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("OVERRIDE_ENV", "New value"),
							tb.EnvVar("OVERRIDE_STAGE_ENV", "New value"),
							tb.EnvVar("SOME_OTHER_VAR", "A value for the other env var"),
							tb.EnvVar("SOME_VAR", "A value for the env var"),
						),
					)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
			),
		},
		{
			name: "dir_on_pipeline_and_stage",
			expected: syntax_helpers_test.ParsedPipeline(
				syntax_helpers_test.PipelineAgent("some-image"),
				syntax_helpers_test.PipelineDir("a-relative-dir"),
				syntax_helpers_test.PipelineStage("A Working Stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("hello"), syntax_helpers_test.StepArg("world")),
				),
				syntax_helpers_test.PipelineStage("Another stage",
					syntax_helpers_test.StageDir("/an/absolute/dir"),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("again")),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("in another dir"),
						syntax_helpers_test.StepDir("another-relative-dir/with/a/subdir"))),
			),
			pipeline: tb.Pipeline("somepipeline-1", "jx", tb.PipelineSpec(
				tb.PipelineTask("a-working-stage", "somepipeline-a-working-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline"),
					tb.PipelineTaskOutputResource("workspace", "somepipeline")),
				tb.PipelineTask("another-stage", "somepipeline-another-stage-1",
					tb.PipelineTaskInputResource("workspace", "somepipeline",
						tb.From("a-working-stage")),
					tb.RunAfter("a-working-stage")),
				tb.PipelineDeclaredResource("somepipeline", tektonv1alpha1.PipelineResourceTypeGit))),
			tasks: []*tektonv1alpha1.Task{
				tb.Task("somepipeline-a-working-stage-1", "jx", syntax_helpers_test.TaskStageLabel("A Working Stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.TaskOutputs(tb.OutputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit)),
					tb.Step("git-merge", syntax.GitMergeImage, tb.Command("jx"), tb.Args("step", "git", "merge", "--verbose"), workingDir("/workspace/source")),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo hello world"), workingDir("/workspace/source/a-relative-dir")),
				)),
				tb.Task("somepipeline-another-stage-1", "jx", syntax_helpers_test.TaskStageLabel("Another stage"), tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", tektonv1alpha1.PipelineResourceTypeGit,
							tb.ResourceTargetPath("source"))),
					tb.Step("step2", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo again"), workingDir("/an/absolute/dir")),
					tb.Step("step3", "some-image", tb.Command("/bin/sh", "-c"), tb.Args("echo in another dir"),
						workingDir("/workspace/source/another-relative-dir/with/a/subdir")),
				)),
			},
			structure: syntax_helpers_test.PipelineStructure("somepipeline-1",
				syntax_helpers_test.StructureStage("A Working Stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-a-working-stage-1")),
				syntax_helpers_test.StructureStage("Another stage", syntax_helpers_test.StructureStageTaskRef("somepipeline-another-stage-1"),
					syntax_helpers_test.StructureStagePrevious("A Working Stage")),
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

			pipeline, tasks, structure, err := parsed.GenerateCRDs("somepipeline", "1", "jx", nil, nil, "source", nil)

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
		expectedError error
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
		{
			name:          "unknown_field",
			expectedError: errors.New("Validation failures in YAML file test_data/validation_failures/unknown_field/jenkins-x.yml:\npipelineConfig: Additional property banana is not allowed"),
		},
		{
			name: "comment_field",
			expectedError: (&apis.FieldError{
				Message: "the comment field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
				Paths:   []string{"comment"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "groovy_field",
			expectedError: (&apis.FieldError{
				Message: "the groovy field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
				Paths:   []string{"groovy"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "when_field",
			expectedError: (&apis.FieldError{
				Message: "the when field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
				Paths:   []string{"when"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "container_field",
			expectedError: (&apis.FieldError{
				Message: "the container field is deprecated - please use image instead",
				Paths:   []string{"container"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "legacy_steps_field",
			expectedError: (&apis.FieldError{
				Message: "the steps field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it and list the nested stages sequentially instead.",
				Paths:   []string{"steps"},
			}).ViaFieldIndex("steps", 0).ViaFieldIndex("stages", 0),
		},
		{
			name: "agent_dir_field",
			expectedError: (&apis.FieldError{
				Message: "the dir field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
				Paths:   []string{"dir"},
			}).ViaField("agent"),
		},
		{
			name: "agent_container_field",
			expectedError: (&apis.FieldError{
				Message: "the container field is deprecated - please use image instead",
				Paths:   []string{"container"},
			}).ViaField("agent"),
		},
		{
			name: "duplicate_step_names",
			expectedError: (&apis.FieldError{
				Message: "step names within a stage must be unique",
				Details: "The following step names in the stage A Working Stage are used more than once: A Step With Spaces And Such, Another Step Name",
				Paths:   []string{"steps"},
			}).ViaFieldIndex("stages", 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConfig, fn, err := config.LoadProjectConfig(filepath.Join("test_data", "validation_failures", tt.name))
			if err != nil && err.Error() != tt.expectedError.Error() {
				t.Fatalf("Failed to parse YAML for %s: %q", tt.name, err)
			}
			if _, ok := tt.expectedError.(*apis.FieldError); ok {

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

func TestParsedPipelineHelpers(t *testing.T) {
	input := syntax_helpers_test.ParsedPipeline(
		syntax_helpers_test.PipelineAgent("some-image"),
		syntax_helpers_test.PipelineOptions(
			syntax_helpers_test.PipelineOptionsRetry(5),
			syntax_helpers_test.PipelineOptionsTimeout(30, syntax.TimeoutUnitSeconds),
		),
		syntax_helpers_test.PipelineEnvVar("ANIMAL", "MONKEY"),
		syntax_helpers_test.PipelineEnvVar("FRUIT", "BANANA"),
		syntax_helpers_test.PipelinePost(syntax.PostConditionSuccess,
			syntax_helpers_test.PostAction("mail", map[string]string{
				"to":      "foo@bar.com",
				"subject": "Yay, it passed",
			})),
		syntax_helpers_test.PipelinePost(syntax.PostConditionFailure,
			syntax_helpers_test.PostAction("slack", map[string]string{
				"whatever": "the",
				"slack":    "config",
				"actually": "is. =)",
			})),
		syntax_helpers_test.PipelineStage("A Working Stage",
			syntax_helpers_test.StageOptions(
				syntax_helpers_test.StageOptionsWorkspace(customWorkspace),
				syntax_helpers_test.StageOptionsStash("some-name", "**/*"),
				syntax_helpers_test.StageOptionsUnstash("some-name", ""),
				syntax_helpers_test.StageOptionsTimeout(15, syntax.TimeoutUnitMinutes),
				syntax_helpers_test.StageOptionsRetry(2),
			),
			syntax_helpers_test.StageStep(
				syntax_helpers_test.StepCmd("echo"),
				syntax_helpers_test.StepArg("hello"),
				syntax_helpers_test.StepArg("world"),
			),
		),
		syntax_helpers_test.PipelineStage("Parent Stage",
			syntax_helpers_test.StageParallel("First Nested Stage",
				syntax_helpers_test.StageAgent("some-other-image"),
				syntax_helpers_test.StageStep(
					syntax_helpers_test.StepCmd("echo"),
					syntax_helpers_test.StepArg("hello"),
					syntax_helpers_test.StepArg("world"),
					syntax_helpers_test.StepAgent("some-other-image"),
				),
				syntax_helpers_test.StageEnvVar("STAGE_VAR_ONE", "some value"),
				syntax_helpers_test.StageEnvVar("STAGE_VAR_TWO", "some other value"),
				syntax_helpers_test.StagePost(syntax.PostConditionAlways,
					syntax_helpers_test.PostAction("junit", map[string]string{
						"pattern": "target/surefire-reports/**/*.xml",
					}),
				),
			),
			syntax_helpers_test.StageParallel("Nested In Parallel",
				syntax_helpers_test.StageSequential("Another stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepLoop("SOME_VAR", []string{"a", "b", "c"},
							syntax_helpers_test.LoopStep(
								syntax_helpers_test.StepCmd("echo"),
								syntax_helpers_test.StepArg("SOME_VAR is ${SOME_VAR}"),
							),
						),
					),
				),
				syntax_helpers_test.StageSequential("Some other stage",
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepCmd("echo"),
						syntax_helpers_test.StepArg("otherwise"),
						syntax_helpers_test.StepDir(customWorkspace),
					),
					syntax_helpers_test.StageStep(
						syntax_helpers_test.StepStep("some-step"),
						syntax_helpers_test.StepOptions(map[string]string{"first": "arg", "second": "arg"}),
					),
				),
			),
		),
	)

	expected := &syntax.ParsedPipeline{
		Agent: &syntax.Agent{
			Image: "some-image",
		},
		Options: &syntax.RootOptions{
			Retry: 5,
			Timeout: &syntax.Timeout{
				Time: 30,
				Unit: syntax.TimeoutUnitSeconds,
			},
		},
		Env: []syntax.EnvVar{
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
				Options: &syntax.StageOptions{
					Workspace: &customWorkspace,
					Stash: &syntax.Stash{
						Name:  "some-name",
						Files: "**/*",
					},
					Unstash: &syntax.Unstash{
						Name: "some-name",
					},
					RootOptions: &syntax.RootOptions{
						Timeout: &syntax.Timeout{
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
						Agent: &syntax.Agent{
							Image: "some-other-image",
						},
						Steps: []syntax.Step{{
							Command:   "echo",
							Arguments: []string{"hello", "world"},
							Agent: &syntax.Agent{
								Image: "some-other-image",
							},
						}},
						Env: []syntax.EnvVar{
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
									Loop: &syntax.Loop{
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
