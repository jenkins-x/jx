package boot

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
)

// BootOptions options for the command
type BootOptions struct {
	*opts.CommonOptions

	Dir string
}

var (
	bootLong = templates.LongDesc(`
		Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline

`)

	bootExample = templates.Examples(`
		# create a kubernetes cluster via Terraform or via jx
		jx create cluster gke --skip-installation

		# lets get the GitOps repository source code
		git clone https://github.com/jstrachan/environment-simple-tekton.git my-jx-config
		cd my-jx-config

		# now lets boot up Jenkins X installing/upgrading whatever is needed
		jx boot 
`)
)

// NewCmdBoot creates the command
func NewCmdBoot(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &BootOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "boot",
		Aliases: []string{"bootstrap"},
		Short:   "Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline",
		Long:    bootLong,
		Example: bootExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the Jenkins X Pipeline, requirements and charts")
	return cmd
}

// Run runs this command
func (o *BootOptions) Run() error {
	requirements, requirementsFile, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err := util.FileExists(requirementsFile)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("No requirements file %s are you sure you are running this command inside a GitOps clone?", requirementsFile)
	}

	err = o.verifyRequirements(requirements, requirementsFile)
	if err != nil {
		return err
	}
	projectConfig, pipelineFile, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err = util.FileExists(requirementsFile)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("No pipeline file %s are you sure you are running this command inside a GitOps clone?", pipelineFile)
	}

	log.Logger().Infof("booting up Jenkins X\n")

	// now lets really boot
	_, so := create.NewCmdStepCreateTaskAndOption(o.CommonOptions)
	so.InterpretMode = true
	so.NoReleasePrepare = true
	so.AdditionalEnvVars = map[string]string{
		"JX_NO_TILLER": "true",
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to interpret pipeline file %s", pipelineFile)
	}

	// if we can find the deploy namespace lets switch kubernetes context to it so the user can use `jx` commands immediately
	ns := FindBootNamespace(projectConfig, requirements)
	if ns != "" {
		info := util.ColorInfo
		no := &namespace.NamespaceOptions{}
		no.CommonOptions = o.CommonOptions
		no.Args = []string{ns}
		log.Logger().Infof("switching to the namespace %s so that you can use %s commands on the installation\n", info(ns), info("jx"))
		return no.Run()
	}
	return nil
}

func (o *BootOptions) verifyRequirements(requirements *config.RequirementsConfig, requirementsFile string) error {
	provider := requirements.Provider
	if provider == "" {
		return o.missingRequirement("provider", requirementsFile)
	}
	if provider == "" {
		if requirements.ProjectID == "" {
			return o.missingRequirement("project", requirementsFile)
		}
	}
	return nil
}

func (o *BootOptions) missingRequirement(property string, fileName string) error {
	return fmt.Errorf("missing property: %s in file %s", property, fileName)
}

// FindBootNamespace finds the namespace to boot Jenkins X into based on the pipeline and requirements
func FindBootNamespace(projectConfig *config.ProjectConfig, requirementsConfig *config.RequirementsConfig) string {
	// TODO should we add the deploy namepace to jx-requirements.yml?
	if projectConfig != nil {
		pipelineConfig := projectConfig.PipelineConfig
		if pipelineConfig != nil {
			release := pipelineConfig.Pipelines.Release
			if release != nil {
				pipeline := release.Pipeline
				if pipeline != nil {
					for _, env := range pipeline.Environment {
						if env.Name == "DEPLOY_NAMESPACE" && env.Value != "" {
							return env.Value
						}
					}
				}
			}
		}
	}
	return ""
}
