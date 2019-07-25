package create_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/stretchr/testify/assert"
)

func TestGetServerlessQuickstartsReturnsTheList(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	defer func() { create.ExecCommand = exec.Command }()
	// Execute
	actual, _ := create.GetServerlessQuickstarts()
	// Evaluate
	assert.Equal(t, 4, len(actual.Quickstarts))
	expectedList := []string{"aws-clojure-gradle", "aws-nodejs", "kubeless-nodejs", "hello-world"}
	for _, expected := range expectedList {
		assert.Contains(t, actual.Quickstarts, expected)
	}
}

func TestGetServerlessQuickstartsReturnsErrorWhenTemplateLineIsNotFound(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpOutputOrig := serverlessCreateHelpOutput
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpOutput = serverlessCreateHelpOutputOrig
	}()
	serverlessCreateHelpOutput = `--something-else`
	// Execute
	_, actual := create.GetServerlessQuickstarts()
	// Evaluate
	assert.Error(t, actual)
}

func TestGetServerlessQuickstartsReturnsErrorWhenNoAvailableTemplatesStringIsMissing(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpOutputOrig := serverlessCreateHelpOutput
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpOutput = serverlessCreateHelpOutputOrig
	}()
	serverlessCreateHelpOutput = `--template / -t .................... Template for the service: "aws-clojure-gradle", "aws-clojurescript-gradle", "kubeless-nodejs", "plugin" and "hello-world"`
	// Execute
	_, actual := create.GetServerlessQuickstarts()
	// Evaluate
	assert.Error(t, actual)
}

func TestGetServerlessQuickstartsReturnsErrorWhenCommandExecutionFails(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpExitCode = "1"
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpExitCode = "0"
	}()
	// Execute
	_, actual := create.GetServerlessQuickstarts()
	// Evaluate
	assert.Error(t, actual)
}

func TestGetServerlessQuickstartsReturnsQuickstarts(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	defer func() { create.ExecCommand = exec.Command }()
	// Execute
	actualList, _ := create.GetServerlessQuickstarts()
	// Evaluate
	// Evaluate one part template
	assert.NotContains(t, actualList.Quickstarts, "plugin")
	// Evaluate two parts template
	assert.Contains(t, actualList.Quickstarts, "aws-nodejs")
	actual := actualList.Quickstarts["aws-nodejs"]
	assert.Equal(t, "aws-nodejs", actual.ID)
	assert.Equal(t, "aws-nodejs", actual.Name)
	assert.Equal(t, "aws", actual.Framework)
	assert.Equal(t, "nodejs", actual.Language)
	// Evaluate two+ parts template
	assert.Contains(t, actualList.Quickstarts, "aws-clojure-gradle")
	actual = actualList.Quickstarts["aws-clojure-gradle"]
	assert.Equal(t, "aws-clojure-gradle", actual.ID)
	assert.Equal(t, "aws-clojure-gradle", actual.Name)
	assert.Equal(t, "aws", actual.Framework)
	assert.Equal(t, "clojure-gradle", actual.Language)
}

func TestCreateServerlessQuickstartExecutesTheCommand(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	actualCommand := ""
	actualArgs := []string{}
	create.ExecCommand = func(command string, args ...string) *exec.Cmd {
		actualCommand = command
		actualArgs = args
		cmd := &exec.Cmd{}
		return cmd
	}
	defer func() { create.ExecCommand = exec.Command }()
	q := quickstarts.Quickstart{
		ID: "my-template",
	}
	qf := quickstarts.QuickstartForm{
		Quickstart: &q,
		Name:       "my-project",
	}
	// Execute
	create.CreateServerlessQuickstart(&qf, "")
	// Validate
	expectedArgs := []string{"create", "--template", q.ID, "--name", qf.Name, "--path", qf.Name}
	assert.Equal(t, "serverless", actualCommand)
	assert.Equal(t, expectedArgs, actualArgs)
}

func TestCreateServerlessQuickstartReturnsErrorWhenCommandExecutionFails(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpExitCode = "1"
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpExitCode = "0"
	}()
	q := quickstarts.Quickstart{
		ID: "my-template",
	}
	qf := quickstarts.QuickstartForm{
		Quickstart: &q,
		Name:       "my-project",
	}
	// Execute
	actual := create.CreateServerlessQuickstart(&qf, "")
	// Evaluate
	assert.Error(t, actual)
}

// Fakes

var execCommandTest = ""
var serverlessCreateHelpExitCode = "0"
var serverlessCreateHelpOutput = `Plugin: Create
create ........................ Create new Serverless service
	--template / -t .................... Template for the service. Available templates: "aws-clojure-gradle", "aws-nodejs", "kubeless-nodejs", "plugin" and "hello-world"
	--template-url / -u ................ Template URL for the service. Supports: GitHub, BitBucket
	--template-path .................... Template local path for the service.
	--path / -p ........................ The path where the service should be created (e.g. --path my-service)
	--name / -n ........................ Name for the service. Overwrites the default name of the created service.`

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=" + execCommandTest, "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{
		"HELPER_PROCESS=1",
		"HELPER_PROCESS_EXIT_CODE=" + serverlessCreateHelpExitCode,
		"HELPER_PROCESS_OUTPUT=" + serverlessCreateHelpOutput,
	}
	return cmd
}

func TestServerlessCommand(t *testing.T) {
	if os.Getenv("HELPER_PROCESS") != "1" {
		return
	}
	exitCode, _ := strconv.Atoi(os.Getenv("HELPER_PROCESS_EXIT_CODE"))
	fmt.Fprintf(os.Stdout, os.Getenv("HELPER_PROCESS_OUTPUT"))
	os.Exit(exitCode)
}
