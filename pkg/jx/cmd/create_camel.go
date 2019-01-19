package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

var (
	createCamelLong = templates.LongDesc(`
		Creates a new Apache Camel application using Spring Boot and then optionally sets up CI/CD pipelines and GitOps promotion.

		For more documentation about Camel see: [https://camel.apache.org/](https://camel.apache.org/)

` + SeeAlsoText("jx create project"))

	createCamelExample = templates.Examples(`
		# Create a Camel application and be prompted for the folder name
		jx create camel 

		# Create a Camel application called awesome
		jx create camel -a awesome
	`)
)

// CreateCamelOptions the options for the create spring command
type CreateCamelOptions struct {
	CreateArchetypeOptions
}

// NewCmdCreateCamel creates a command object for the "create" command
func NewCmdCreateCamel(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateCamelOptions{
		CreateArchetypeOptions{
			CreateProjectOptions: CreateProjectOptions{
				ImportOptions: ImportOptions{
					CommonOptions: CommonOptions{
						Factory: f,
						In:      in,

						Out: out,
						Err: errOut,
					},
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "camel",
		Short:   "Create a new Camel based application and import the generated code into Git and Jenkins for CI/CD",
		Long:    createCamelLong,
		Example: createCamelExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Form.ArchetypeVersion, "camel-version", "c", "RELEASE", "The Version of the Archetype to use")
	options.addCreateAppFlags(cmd)
	options.addGeneratedMvnCoordinateFlags(cmd)

	return cmd
}

// Run implements the command
func (o *CreateCamelOptions) Run() error {
	o.Form.ArchetypeGroupId = "org.apache.camel.archetypes"
	o.Form.ArchetypeArtifactId = "camel-archetype-spring-boot"

	return o.CreateArchetype()
}
