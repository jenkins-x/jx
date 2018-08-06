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

package clone

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/kube"
)

// Run clones the refs under the prescribed directory and optionally
// configures the git username and email in the repository as well.
func Run(refs *kube.Refs, dir, gitUserName, gitUserEmail string, env []string) Record {
	logrus.WithFields(logrus.Fields{"refs": refs}).Info("Cloning refs")
	record := Record{Refs: refs}
	repositoryURI := fmt.Sprintf("https://github.com/%s/%s.git", refs.Org, refs.Repo)
	if refs.CloneURI != "" {
		repositoryURI = refs.CloneURI
	}
	cloneDir := PathForRefs(dir, refs)

	commands := []cloneCommand{
		func() (string, string, error) {
			return fmt.Sprintf("os.MkdirAll(%s, 0755)", cloneDir), "", os.MkdirAll(cloneDir, 0755)
		},
	}

	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "init"))
	if gitUserName != "" {
		commands = append(commands, shellCloneCommand(cloneDir, env, "git", "config", "user.name", gitUserName))
	}
	if gitUserEmail != "" {
		commands = append(commands, shellCloneCommand(cloneDir, env, "git", "config", "user.email", gitUserEmail))
	}
	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "fetch", repositoryURI, "--tags", "--prune"))
	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "fetch", repositoryURI, refs.BaseRef))

	var target string
	if refs.BaseSHA != "" {
		target = refs.BaseSHA
	} else {
		target = "FETCH_HEAD"
	}
	// we need to be "on" the target branch after the sync
	// so we need to set the branch to point to the base ref,
	// but we cannot update a branch we are on, so in case we
	// are on the branch we are syncing, we check out the SHA
	// first and reset the branch second, then check out the
	// branch we just reset to be in the correct final state
	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "checkout", target))
	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "branch", "--force", refs.BaseRef, target))
	commands = append(commands, shellCloneCommand(cloneDir, env, "git", "checkout", refs.BaseRef))

	for _, prRef := range refs.Pulls {
		ref := fmt.Sprintf("pull/%d/head", prRef.Number)
		if prRef.Ref != "" {
			ref = prRef.Ref
		}
		commands = append(commands, shellCloneCommand(cloneDir, env, "git", "fetch", repositoryURI, ref))
		var prCheckout string
		if prRef.SHA != "" {
			prCheckout = prRef.SHA
		} else {
			prCheckout = "FETCH_HEAD"
		}
		commands = append(commands, shellCloneCommand(cloneDir, env, "git", "merge", prCheckout))
	}

	for _, command := range commands {
		formattedCommand, output, err := command()
		logrus.WithFields(logrus.Fields{"command": formattedCommand, "output": output, "error": err}).Info("Ran command")
		message := ""
		if err != nil {
			message = err.Error()
			record.Failed = true
		}
		record.Commands = append(record.Commands, Command{Command: formattedCommand, Output: output, Error: message})
		if err != nil {
			break
		}
	}

	return record
}

// PathForRefs determines the full path to where
// refs should be cloned
func PathForRefs(baseDir string, refs *kube.Refs) string {
	var clonePath string
	if refs.PathAlias != "" {
		clonePath = refs.PathAlias
	} else {
		clonePath = fmt.Sprintf("github.com/%s/%s", refs.Org, refs.Repo)
	}
	return fmt.Sprintf("%s/src/%s", baseDir, clonePath)
}

type cloneCommand func() (string, string, error)

func shellCloneCommand(dir string, env []string, command string, args ...string) cloneCommand {
	output := bytes.Buffer{}
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdout = &output
	cmd.Stderr = &output

	return func() (string, string, error) {
		err := cmd.Run()
		return strings.Join(append([]string{command}, args...), " "), output.String(), err
	}
}
