package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/spring"
)

var (
	create_spring_long = templates.LongDesc(`
		Creates a new Spring Boot application on the file system.

		You then get the option to import the source code into a git repository and Jenkins for CI / CD

`)

	create_spring_example = templates.Examples(`
		# Create a Spring Boot application where you use the terminal to pick the values
		jx create spring

		# Creates a Spring Boot application passing in the required dependencies
		jx create spring -d web,actuator
	`)
)

// CreateSpringOptions the options for the create spring command
type CreateSpringOptions struct {
	CreateOptions

	// spring boot
	Advanced   bool
	SpringForm spring.SpringBootForm
	OutDir     string
}

// NewCmdCreateSpring creates a command object for the "create" command
func NewCmdCreateSpring(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateSpringOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "spring",
		Short:   "Create a new spring boot application and import it into git and Jenkins for CI / CD",
		Long:    create_spring_long,
		Example: create_spring_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Advanced, "advanced", "x", false, "Advanced mode can show more detailed forms for some resource kinds like springboot")
	cmd.Flags().BoolVarP(&options.DisableImport, "no-import", "c", false, "Disable import after the creation")

	// spring flags
	cmd.Flags().StringArrayVarP(&options.SpringForm.DependencyKinds, spring.OptionDependencyKind, "k", spring.DefaultDependencyKinds, "Default dependency kinds to choose from")
	cmd.Flags().StringArrayVarP(&options.SpringForm.Dependencies, spring.OptionDependency, "d", []string{}, "Spring Boot dependencies")
	cmd.Flags().StringVarP(&options.SpringForm.GroupId, spring.OptionGroupId, "g", "", "Group ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.ArtifactId, spring.OptionArtifactId, "a", "", "Artifact ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.Language, spring.OptionLanguage, "l", "", "Language to generate")
	cmd.Flags().StringVarP(&options.SpringForm.BootVersion, spring.OptionBootVersion, "b", "", "Spring Boot version")
	cmd.Flags().StringVarP(&options.SpringForm.JavaVersion, spring.OptionJavaVersion, "j", "", "Java version")
	cmd.Flags().StringVarP(&options.SpringForm.Packaging, spring.OptionPackaging, "p", "", "Packaging")

	cmd.Flags().StringVarP(&options.OutDir, "output-dir", "o", "", "Directory to output the project to. Defaults to the current directory")
	return cmd
}

// Run implements the generic Create command
func (o *CreateSpringOptions) Run() error {
	cacheDir, err := cmdutil.CacheDir()
	if err != nil {
		return err
	}
	model, err := spring.LoadSpringBoot(cacheDir)
	if err != nil {
		return fmt.Errorf("Failed to load Spring Boot model %s", err)
	}

	data := &o.SpringForm
	err = model.CreateSurvey(&o.SpringForm, o.Advanced)
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
	o.Printf("Created spring boot project at %s\n", outDir)

	return o.DoImport(outDir)
}
