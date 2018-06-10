package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// CreateOptions contains the command line options
type CreateOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

// CreateProjectOptions contains the command line options
type CreateProjectOptions struct {
	ImportOptions

	DisableImport bool
	OutDir        string
}

var (
	create_resources = `Valid resource types include:

	* archetype
	* cluster
	* env
	* git
	* spring (aka 'springboot')
    `

	create_long = templates.LongDesc(`
		Creates a new resource.

		` + create_resources + `
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
		Use:   "create",
		Short: "Create a new resource",
		Long:  create_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateAddon(f, out, errOut))
	cmd.AddCommand(NewCmdCreateArchetype(f, out, errOut))
	cmd.AddCommand(NewCmdCreateBranchPattern(f, out, errOut))
	cmd.AddCommand(NewCmdCreateCamel(f, out, errOut))
	cmd.AddCommand(NewCmdCreateChat(f, out, errOut))
	cmd.AddCommand(NewCmdCreateCluster(f, out, errOut))
	cmd.AddCommand(NewCmdCreateDevPod(f, out, errOut))
	cmd.AddCommand(NewCmdCreateDocs(f, out, errOut))
	cmd.AddCommand(NewCmdCreateEnv(f, out, errOut))
	cmd.AddCommand(NewCmdCreateEtcHosts(f, out, errOut))
	cmd.AddCommand(NewCmdCreateGit(f, out, errOut))
	cmd.AddCommand(NewCmdCreateIssue(f, out, errOut))
	cmd.AddCommand(NewCmdCreateJenkins(f, out, errOut))
	cmd.AddCommand(NewCmdCreateJHipster(f, out, errOut))
	cmd.AddCommand(NewCmdCreateMicro(f, out, errOut))
	cmd.AddCommand(NewCmdCreateQuickstart(f, out, errOut))
	cmd.AddCommand(NewCmdCreateQuickstartLocation(f, out, errOut))
	cmd.AddCommand(NewCmdCreateSpring(f, out, errOut))
	cmd.AddCommand(NewCmdCreateToken(f, out, errOut))
	cmd.AddCommand(NewCmdCreateTracker(f, out, errOut))
	cmd.AddCommand(NewCmdCreateLile(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateOptions) Run() error {
	return o.Cmd.Help()
}

// DoImport imports the project created at the given directory
func (o *CreateProjectOptions) ImportCreatedProject(outDir string) error {
	if o.DisableImport {
		return nil
	}
	importOptions := &o.ImportOptions
	importOptions.Dir = outDir
	importOptions.DisableDotGitSearch = true
	return importOptions.Run()
}

func (options *CreateProjectOptions) addCreateAppFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&options.DisableImport, "no-import", "", false, "Disable import after the creation")
	cmd.Flags().StringVarP(&options.OutDir, "output-dir", "o", "", "Directory to output the project to. Defaults to the current directory")

	options.addImportFlags(cmd, true)
}
