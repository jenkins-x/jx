package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/maven"
	"os"
	"path/filepath"
)

var (
	create_archetype_long = templates.LongDesc(`
		Creates a new Maven project using an Archetype

		You then get the option to import the generated source code into a git repository and Jenkins for CI / CD

`)

	create_archetype_example = templates.Examples(`
		# Create a new application from a Maven Archetype using the UI to choose which archetype to use
		jx create archetype

		# Creates a Camel Archetype, filtering on the archetypes containing the text 'spring'
		jx create archetype -g  org.apache.camel.archetypes -a spring
	`)
)

// CreateArchetypeOptions the options for the create spring command
type CreateArchetypeOptions struct {
	CreateOptions

	ArchetypeCatalogURL string

	Filter      maven.ArchetypeFilter
	PickVersion bool
	Interactive bool

	Form maven.ArchetypeForm
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
		Short:   "Create a new app from a Maven Archetype and import the generated code into git and Jenkins for CI / CD",
		Long:    create_archetype_long,
		Example: create_archetype_example,
		Aliases: []string{"arch"},
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
	cmd.Flags().BoolVarP(&options.Interactive, "interactive", "i", false, "Allow interactive input into the maven archetype:generate command")

	cmd.Flags().StringVarP(&options.Form.GroupId, "create-group", "", "", "The group ID for the new application")
	cmd.Flags().StringVarP(&options.Form.ArtifactId, "create-artifact", "n", "", "The artifact ID for the new application")
	cmd.Flags().StringVarP(&options.Form.Version, "create-version", "", "", "The version for the new application")
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
	form := &o.Form
	err = model.CreateSurvey(&o.Filter, o.PickVersion, form)
	if err != nil {
		return err
	}

	o.Printf("Invoking: jx create archetype -g %s -a %s -v %s\n\n", form.ArchetypeGroupId, form.ArchetypeArtifactId, form.ArchetypeVersion)

	dir := o.OutDir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	o.Printf("basedir is: %s\n", dir)

	args := []string{}
	if !o.Interactive {
		args = append(args, "-B")
	}
	args = append(args, "org.apache.maven.plugins:maven-archetype-plugin:"+maven.MavenArchetypePluginVersion+":generate",
		"-DarchetypeGroupId="+form.ArchetypeGroupId,
		"-DarchetypeArtifactId="+form.ArchetypeArtifactId,
		"-DarchetypeVersion="+form.ArchetypeVersion,
		"-Dbasedir="+dir)

	if form.GroupId != "" {
		args = append(args, "-DgroupId="+form.GroupId)
	}
	if form.ArtifactId != "" {
		args = append(args, "-DartifactId="+form.ArtifactId)
	}
	if form.Version != "" {
		args = append(args, "-Dversion="+form.Version)
	}

	err = o.runCommandInteractive(o.Interactive, "mvn", args...)
	if err != nil {
		return err
	}
	outDir := filepath.Join(dir, form.ArtifactId)
	o.Printf("Created project at %s\n\n", outDir)

	return o.DoImport(outDir)
}
