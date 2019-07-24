package create_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/stretchr/testify/assert"
)

func TestGetServerlessTemplatesReturnsTheList(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCreateHelpCommand"
	create.ExecCommand = fakeExecCommand
	defer func() { create.ExecCommand = exec.Command }()
	// Execute
	actual, _ := create.GetServerlessTemplates()
	// Evaluate
	assert.Equal(t, 5, len(actual))
	assert.Equal(t, []string{"aws-clojure-gradle", "aws-clojurescript-gradle", "kubeless-nodejs", "plugin", "hello-world"}, actual)
}

func TestGetServerlessTemplatesReturnsErrorWhenTemplateLineIsNotFound(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCreateHelpCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpOutputOrig := serverlessCreateHelpOutput
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpOutput = serverlessCreateHelpOutputOrig
	}()
	serverlessCreateHelpOutput = `--something-else`
	// Execute
	_, actual := create.GetServerlessTemplates()
	// Evaluate
	assert.Error(t, actual)
}

func TestGetServerlessTemplatesReturnsErrorWhenNoAvailableTemplatesStringIsMissing(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCreateHelpCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpOutputOrig := serverlessCreateHelpOutput
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpOutput = serverlessCreateHelpOutputOrig
	}()
	serverlessCreateHelpOutput = `--template / -t .................... Template for the service: "aws-clojure-gradle", "aws-clojurescript-gradle", "kubeless-nodejs", "plugin" and "hello-world"`
	// Execute
	_, actual := create.GetServerlessTemplates()
	// Evaluate
	assert.Error(t, actual)
}

func TestGetServerlessTemplatesReturnsErrorWhenCommandExecutionFails(t *testing.T) {
	// Prepare
	execCommandTest = "TestServerlessCreateHelpCommand"
	create.ExecCommand = fakeExecCommand
	serverlessCreateHelpExitCode = "1"
	defer func() {
		create.ExecCommand = exec.Command
		serverlessCreateHelpExitCode = "0"
	}()
	// Execute
	_, actual := create.GetServerlessTemplates()
	// Evaluate
	assert.Error(t, actual)
}

// Fakes

var execCommandTest = ""
var serverlessCreateHelpExitCode = "0"
var serverlessCreateHelpOutput = `Plugin: Create
create ........................ Create new Serverless service
	--template / -t .................... Template for the service. Available templates: "aws-clojure-gradle", "aws-clojurescript-gradle", "kubeless-nodejs", "plugin" and "hello-world"
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

func TestServerlessCreateHelpCommand(t *testing.T) {
	if os.Getenv("HELPER_PROCESS") != "1" {
		return
	}
	exitCode, _ := strconv.Atoi(os.Getenv("HELPER_PROCESS_EXIT_CODE"))
	fmt.Fprintf(os.Stdout, os.Getenv("HELPER_PROCESS_OUTPUT"))
	os.Exit(exitCode)
}
