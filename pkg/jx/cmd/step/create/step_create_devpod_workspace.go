package create

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/packages"

	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var (
	createDevPodWorkpaceLong = templates.LongDesc(`
		Creates the DevPod workspace files
`)

	createDevPodWorkpaceExample = templates.Examples(`
		# create the DevPod Workspace files
		jx step create devpod workspace

			`)
)

// StepCreateDevPodWorkpaceOptions contains the command line flags
type StepCreateDevPodWorkpaceOptions struct {
	opts.StepOptions

	Dir string
}

// StepCreateDevPodWorkpaceResults stores the generated results
type StepCreateDevPodWorkpaceResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreateDevPodWorkpace Creates a new Command object
func NewCmdStepCreateDevPodWorkpace(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateDevPodWorkpaceOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "devpod workspace",
		Short:   "Creates the DevPod workspace files",
		Long:    createDevPodWorkpaceLong,
		Example: createDevPodWorkpaceExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "/workspace/bin", "The bin directory in the workspace")
	return cmd
}

// Run implements this command
func (o *StepCreateDevPodWorkpaceOptions) Run() error {
	const kubectl = "kubectl"
	path, err := packages.LookupForBinary(kubectl)
	if err != nil {
		return errors.Wrapf(err, "could not find binary %s on the $PATH", kubectl)
	}
	outDir := o.Dir
	if outDir == "" {
		outDir = "."
	}
	err = os.MkdirAll(outDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure output directory is created %s", outDir)
	}

	destPath := filepath.Join(outDir, kubectl)
	if path != destPath {
		exists, err := util.FileExists(destPath)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", destPath)
		}
		if !exists {
			err = util.CopyFile(path, destPath)
			if err != nil {
				return errors.Wrapf(err, "failed to copy %s to %s", path, destPath)
			}
		}
	}

	scriptPath := filepath.Join(outDir, "devpodsh")

	text := fmt.Sprintf(`#!/bin/sh

DIR=$(pwd)
echo "opening shell inside DevPod with args: $* in dir $DIR"
%s/kubectl exec -it -c devpod $HOSTNAME bash -- -c "cd $DIR && bash"
`, outDir)

	err = ioutil.WriteFile(scriptPath, []byte(text), util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save script to %s", scriptPath)
	}
	return nil
}
