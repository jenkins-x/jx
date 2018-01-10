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

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type CreateOptions struct {
	CommonOptions

	DisableImport bool

	// spring boot
	Advanced bool
	SpringForm spring.SpringBootForm
	OutDir string
}

var (
	create_resources = `Valid resource types include:

    * springboot (aka 'spring')
    `

	create_long = templates.LongDesc(`
		Creates one or many resources.

		` + valid_resources + `

`)

	create_example = templates.Examples(`
		# List all pipeines
		kubectl get pipeline

		# List all URLs for services in the current namespace
		kubectl get url
	`)
)

// NewCmdCreate creates a command object for the "create" command
func NewCmdCreate(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "create TYPE [flags]",
		Short:   "Create a new resource",
		Long:    create_long,
		Example: create_example,
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
	cmd.Flags().StringArrayVarP(&options.SpringForm.DependencyKinds, "kind", "k", spring.DefaultDependencyKinds, "Default dependency kinds to choose from")
	cmd.Flags().StringArrayVarP(&options.SpringForm.Dependencies, "dep", "d", []string{}, "Spring Boot dependencies")
	cmd.Flags().StringVarP(&options.SpringForm.GroupId, "group", "g", "", "Group ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.ArtifactId, "artifact", "a", "", "Artifact ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.Language, "language", "l", "", "Language to generate")
	cmd.Flags().StringVarP(&options.SpringForm.BootVersion, "boot-version", "b", "", "Spring Boot version")
	cmd.Flags().StringVarP(&options.SpringForm.Packaging, "packaging", "p", "", "Packaging")

	cmd.Flags().StringVarP(&options.OutDir, "output-dir", "o", "", "Directory to output the project to. Defaults to the current directory")
	return cmd
}

// Run implements the generic Create command
func (options *CreateOptions) Run() error {
	args := options.Args
	cmd := options.Cmd
	if len(args) == 0 {
		fmt.Fprint(options.Err, "You must specify the type of resource to create. ", create_resources)

		usageString := "Required resource not specified."
		return cmdutil.UsageError(cmd, usageString)
	}
	kind := args[0]
	switch kind {
	case "spring":
		return options.createSpringBoot()
	case "springboot":
		return options.createSpringBoot()
	case "spring-boot":
		return options.createSpringBoot()

	default:
		return cmdutil.UsageError(cmd, "Unknown resource kind: %s", kind)
	}
	return nil

}

func (o *CreateOptions) createSpringBoot() error {
	model, err := spring.LoadSpringBoot()
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

	if o.DisableImport {
		return nil
	}

	// now lets import
	importOptions := &ImportOptions{
		CommonOptions: o.CommonOptions,
		Dir: outDir,
	}
	return importOptions.Run()
}
