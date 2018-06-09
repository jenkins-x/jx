package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

var (
	createCamelLong = templates.LongDesc(`
		Creates a new Apache Camel application using Spring Boot and then optionally setups CI/CD pipelines and GitOps promotion.

		For more documentation about camel see: [https://camel.apache.org/](https://camel.apache.org/)

	`)

	createCamelExample = templates.Examples(`
		# Create a camel application and be prompted for the folder name
		jx create camel 

		# Create a camel application called awesome
		jx create camel -a awesome
	`)
)

// CreateCamelOptions the options for the create spring command
type CreateCamelOptions struct {
	CreateArchetypeOptions
}

// NewCmdCreateCamel creates a command object for the "create" command
func NewCmdCreateCamel(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateCamelOptions{
		CreateArchetypeOptions{
			CreateProjectOptions: CreateProjectOptions{
				ImportOptions: ImportOptions{
					CommonOptions: CommonOptions{
						Factory: f,
						Out:     out,
						Err:     errOut,
					},
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "camel",
		Short:   "Create a new camel based application and import the generated code into git and Jenkins for CI/CD",
		Long:    createCamelLong,
		Example: createCamelExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
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
