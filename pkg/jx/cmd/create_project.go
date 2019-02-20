package cmd

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	createQuickstartName = "Create new application from a Quickstart"
	createCamelName      = "Create new Apache Camel microservice"
	createSpringName     = "Create new Spring Boot microservice"
	importDirName        = "Import existing code from a directory"
	importGitName        = "Import code from a git repository"
	importGitHubName     = "Import code from a github repository"
)

var (
	createProjectNames = []string{
		createQuickstartName,
		createCamelName,
		createSpringName,
		importDirName,
		importGitName,
		importGitHubName,
	}

	createProjectLong = templates.LongDesc(`
		Create a new Project by importing code, using a Quickstart or custom wizard for Spring or Camel.

` + SeeAlsoText("jx create quickstart", "jx create spring", "jx create camel", "jx create jhipster", "jx import"))

	createProjectExample = templates.Examples(`
		# Create a project
		jx create project
	`)
)

// CreateProjectWizardOptions the options for the command
type CreateProjectWizardOptions struct {
	CreateOptions
}

// NewCmdCreateProject creates a command object for the "create" command
func NewCmdCreateProject(commonOpts *CommonOptions) *cobra.Command {
	options := &CreateProjectWizardOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "project",
		Short:   "Create a new Project by importing code, using a Quickstart or custom wizard for Spring or Camel",
		Long:    createProjectLong,
		Example: createProjectExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	return cmd
}

// Run implements the command
func (o *CreateProjectWizardOptions) Run() error {
	name, err := util.PickName(createProjectNames, "Which kind of project you want to create: ",
		"Jenkins X supports a number of diffferent wizards for creating or importing new projects.",
		o.In, o.Out, o.Err)
	if err != nil {
		return err
	}
	switch name {
	case createCamelName:
		return o.createCamel()
	case createQuickstartName:
		return o.createQuickstart()
	case createSpringName:
		return o.createSpring()
	case importDirName:
		return o.importDir()
	case importGitName:
		return o.importGit()
	case importGitHubName:
		return o.importGithubProject()
	default:
		return fmt.Errorf("Unknown selection: %s\n", name)
	}
}

func (o *CreateProjectWizardOptions) createCamel() error {
	w := &CreateCamelOptions{}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *CreateProjectWizardOptions) createQuickstart() error {
	w := &CreateQuickstartOptions{}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *CreateProjectWizardOptions) createSpring() error {
	w := &CreateSpringOptions{}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *CreateProjectWizardOptions) importDir() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := util.PickValue("Which directory contains the source code: ", wd, true,
		"Please specify the directory which contains the source code you want to use for your new project", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}
	w := &ImportOptions{
		Dir: dir,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *CreateProjectWizardOptions) importGit() error {
	repoUrl, err := util.PickValue("Which git repository URL to inmport: ", "", true,
		"Please specify the git URL which contains the source code you want to use for your new project", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	w := &ImportOptions{
		RepoURL: repoUrl,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *CreateProjectWizardOptions) importGithubProject() error {
	w := &ImportOptions{
		GitHub: true,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}
