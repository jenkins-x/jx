package cmd

import (
	"fmt"
	"io"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/maven"
)

var (
	create_archetype_long = templates.LongDesc(`
		Creates a new Spring Boot application on the file system.

		You then get the option to import the source code into a git repository and Jenkins for CI / CD

`)

	create_archetype_example = templates.Examples(`
		# Create a Spring Boot application where you use the terminal to pick the values
		jx create spring

		# Creates a Spring Boot application passing in the required dependencies
		jx create spring -d web,actuator
	`)
)

// CreateArchetypeOptions the options for the create spring command
type CreateArchetypeOptions struct {
	CreateOptions

	ArchetypeCatalogURL string

	Filter maven.ArchetypeFilter
	PickVersion  bool
}

// NewCmdCreateArchetype creates a command object for the "create" command
func NewCmdCreateArchetype(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateArchetypeOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "archetype",
		Short:   "Create a new app from a Maven Archetype and import it into git and Jenkins for CI / CD",
		Long:    create_archetype_long,
		Example: create_archetype_example,
		Aliases: []string{"quickstart"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	addCreateFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.ArchetypeCatalogURL, "catalog", "c", "http://central.maven.org/maven2/archetype-catalog.xml", "The Maven Archetype Catalog to use")

	cmd.Flags().StringArrayVarP(&options.Filter.GroupIds, "group", "g", []string{}, "The Group ID of the Archetypes")
	cmd.Flags().StringVarP(&options.Filter.GroupIdFilter, "group-filter", "f", "", "Filter the Group IDs to choose from for he Archetypes")
	cmd.Flags().StringVarP(&options.Filter.ArtifactIdFilter, "artifact", "a", "", "Either the Artifact ID or a text filter of the artifact IDs to pick from")
	cmd.Flags().StringVarP(&options.Filter.Version, "version", "v", "", "The Version of the Archetype to use")
	cmd.Flags().BoolVarP(&options.PickVersion, "pick", "p", false, "Provide a list of versions to choose from")
	return cmd
}

// Run implements the generic Create command
func (o *CreateArchetypeOptions) Run() error {
	cacheDir, err := cmdutil.CacheDir()
	if err != nil {
		return err
	}
	model, err := maven.LoadArchetypes("default", o.ArchetypeCatalogURL, cacheDir)
	if err != nil {
		return fmt.Errorf("Failed to load Spring Boot model %s", err)
	}
	err = model.CreateSurvey(&o.Filter, o.PickVersion)
	if err != nil {
		return err
	}

	return nil

	/*
	outDir, err := data.CreateProject(dir)
	if err != nil {
		return err
	}
	o.Printf("Created spring boot project at %s\n", outDir)

	return o.DoImport(outDir)
	*/
}
