package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/spring"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	create_spring_long = templates.LongDesc(`
		Creates a new Spring Boot application and then optionally setups CI/CD pipelines and GitOps promotion.

		You can see a demo of this command here: [http://jenkins-x.io/demos/create_spring/](http://jenkins-x.io/demos/create_spring/)

		For more documentation see: [http://jenkins-x.io/developing/create-spring/](http://jenkins-x.io/developing/create-spring/)

`)

	create_spring_example = templates.Examples(`
		# Create a Spring Boot application where you use the terminal to pick the values
		jx create spring

		# Creates a Spring Boot application passing in the required dependencies
		jx create spring -d web -d actuator

		# To pick the advanced options (such as what package type maven-project/gradle-project) etc then use
		jx create spring -x

		# To create a gradle project use:
		jx create spring --type gradle-project
	`)
)

// CreateSpringOptions the options for the create spring command
type CreateSpringOptions struct {
	CreateProjectOptions

	Advanced   bool
	SpringForm spring.SpringBootForm
}

// NewCmdCreateSpring creates a command object for the "create" command
func NewCmdCreateSpring(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateSpringOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "spring",
		Short:   "Create a new spring boot application and import the generated code into git and Jenkins for CI/CD",
		Long:    create_spring_long,
		Example: create_spring_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCreateAppFlags(cmd)

	cmd.Flags().BoolVarP(&options.Advanced, "advanced", "x", false, "Advanced mode can show more detailed forms for some resource kinds like springboot")

	cmd.Flags().StringArrayVarP(&options.SpringForm.DependencyKinds, spring.OptionDependencyKind, "k", spring.DefaultDependencyKinds, "Default dependency kinds to choose from")
	cmd.Flags().StringArrayVarP(&options.SpringForm.Dependencies, spring.OptionDependency, "d", []string{}, "Spring Boot dependencies")
	cmd.Flags().StringVarP(&options.SpringForm.GroupId, spring.OptionGroupId, "g", "", "Group ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.ArtifactId, spring.OptionArtifactId, "a", "", "Artifact ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.Language, spring.OptionLanguage, "l", "", "Language to generate")
	cmd.Flags().StringVarP(&options.SpringForm.BootVersion, spring.OptionBootVersion, "t", "", "Spring Boot version")
	cmd.Flags().StringVarP(&options.SpringForm.JavaVersion, spring.OptionJavaVersion, "j", "", "Java version")
	cmd.Flags().StringVarP(&options.SpringForm.Packaging, spring.OptionPackaging, "p", "", "Packaging")
	cmd.Flags().StringVarP(&options.SpringForm.Type, spring.OptionType, "", "", "Project Type (such as maven-project or gradle-project)")

	return cmd
}

// Run implements the command
func (o *CreateSpringOptions) Run() error {
	cacheDir, err := util.CacheDir()
	if err != nil {
		return err
	}
	model, err := spring.LoadSpringBoot(cacheDir)
	if err != nil {
		return fmt.Errorf("Failed to load Spring Boot model %s", err)
	}

	data := &o.SpringForm
	err = model.CreateSurvey(&o.SpringForm, o.Advanced, o.BatchMode)
	if err != nil {
		return err
	}

	dir := o.OutDir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	outDir, err := data.CreateProject(dir)
	if err != nil {
		return err
	}
	o.Printf("Created spring boot project at %s\n", util.ColorInfo(outDir))

	return o.ImportCreatedProject(outDir)
}
