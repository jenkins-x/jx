package tasks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	ErrNoTaskFile = errors.New(".draft-tasks.toml not found")
)

const (
	PreUp      = "PreUp"
	PostDeploy = "PostDeploy"
	PostDelete = "PostDelete"
)

type Tasks struct {
	PreUp      map[string]string `toml:"pre-up"`
	PostDeploy map[string]string `toml:"post-deploy"`
	PostDelete map[string]string `toml:"cleanup"`
}

type Result struct {
	Kind    string
	Command []string
	Pass    bool
	Message string
}

// Load takes a path to file where tasks are defined and loads them in tasks
func Load(path string) (*Tasks, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoTaskFile
		}
		return nil, err
	}

	t := Tasks{}
	if _, err := toml.DecodeFile(path, &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (t *Tasks) Run(kind, podName string) ([]Result, error) {
	results := []Result{}

	switch kind {
	case PreUp:
		for _, task := range t.PreUp {
			result := executeTask(task, kind)
			results = append(results, result)
		}
	case PostDeploy:
		for _, task := range t.PostDeploy {
			cmd := preparePostDeployTask(evaluateArgs(task), podName)
			result := runTask(cmd, kind)
			results = append(results, result)
		}
	case PostDelete:
		for _, task := range t.PostDelete {
			result := executeTask(task, kind)
			results = append(results, result)
		}
	default:
		return results, fmt.Errorf("Task kind: %s not supported", kind)
	}

	return results, nil
}

func executeTask(task, kind string) Result {
	args := evaluateArgs(task)
	cmd := prepareTask(args)
	return runTask(cmd, kind)
}

func runTask(cmd *exec.Cmd, kind string) Result {
	result := Result{Kind: kind, Pass: false}
	result.Command = append([]string{cmd.Path}, cmd.Args[0:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		result.Pass = false
		result.Message = err.Error()
		return result
	}
	result.Pass = true

	return result
}

func prepareTask(args []string) *exec.Cmd {
	var cmd *exec.Cmd
	if len(args) < 2 {
		cmd = exec.Command(args[0])
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}
	return cmd
}

func preparePostDeployTask(args []string, podName string) *exec.Cmd {
	args = append([]string{"exec", podName, "--"}, args[0:]...)
	return exec.Command("kubectl", args[0:]...)
}

func evaluateArgs(task string) []string {
	args := strings.Split(task, " ")
	argsCopy := args
	count := 0
	for _, part := range argsCopy {
		if strings.HasPrefix(part, "$") {
			evaluatedPart := os.Getenv(strings.TrimPrefix(part, "$"))
			args[count] = evaluatedPart
		}
		count = count + 1
	}
	return args
}
