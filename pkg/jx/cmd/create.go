package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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
func NewCmdCreate(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
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
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateAddon(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateArchetype(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateBranchPattern(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateCamel(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateChat(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateCodeship(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateCluster(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateDevPod(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateDockerAuth(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateDocs(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateEtcHosts(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateGkeServiceAccount(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateGit(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateIssue(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateJenkins(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateJHipster(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateLile(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateMicro(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreatePostPreviewJob(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateProject(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreatePullRequest(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateQuickstart(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateQuickstartLocation(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateSpring(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateTeam(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateTerraform(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateToken(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateTracker(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateUser(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateVault(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateClient(f, in, out, errOut))

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
