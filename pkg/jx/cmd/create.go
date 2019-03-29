package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// CreateOptions contains the command line options
type CreateOptions struct {
	*CommonOptions

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
func NewCmdCreate(commonOpts *CommonOptions) *cobra.Command {
	options := &CreateOptions{
		CommonOptions: commonOpts,
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

	cmd.AddCommand(NewCmdCreateAddon(commonOpts))
	cmd.AddCommand(NewCmdCreateArchetype(commonOpts))
	cmd.AddCommand(NewCmdCreateBranchPattern(commonOpts))
	cmd.AddCommand(NewCmdCreateCamel(commonOpts))
	cmd.AddCommand(NewCmdCreateChat(commonOpts))
	cmd.AddCommand(NewCmdCreateCodeship(commonOpts))
	cmd.AddCommand(NewCmdCreateCluster(commonOpts))
	cmd.AddCommand(NewCmdCreateDevPod(commonOpts))
	cmd.AddCommand(NewCmdCreateDockerAuth(commonOpts))
	cmd.AddCommand(NewCmdCreateDocs(commonOpts))
	cmd.AddCommand(NewCmdCreateEnv(commonOpts))
	cmd.AddCommand(NewCmdCreateEtcHosts(commonOpts))
	cmd.AddCommand(NewCmdCreateGkeServiceAccount(commonOpts))
	cmd.AddCommand(NewCmdCreateGit(commonOpts))
	cmd.AddCommand(NewCmdCreateIssue(commonOpts))
	cmd.AddCommand(NewCmdCreateJenkins(commonOpts))
	cmd.AddCommand(NewCmdCreateJHipster(commonOpts))
	cmd.AddCommand(NewCmdCreateLile(commonOpts))
	cmd.AddCommand(NewCmdCreateMicro(commonOpts))
	cmd.AddCommand(NewCmdCreatePostPreviewJob(commonOpts))
	cmd.AddCommand(NewCmdCreateProject(commonOpts))
	cmd.AddCommand(NewCmdCreatePullRequest(commonOpts))
	cmd.AddCommand(NewCmdCreateQuickstart(commonOpts))
	cmd.AddCommand(NewCmdCreateQuickstartLocation(commonOpts))
	cmd.AddCommand(NewCmdCreateSpring(commonOpts))
	cmd.AddCommand(NewCmdCreateStep(commonOpts))
	cmd.AddCommand(NewCmdCreateTeam(commonOpts))
	cmd.AddCommand(NewCmdCreateTerraform(commonOpts))
	cmd.AddCommand(NewCmdCreateToken(commonOpts))
	cmd.AddCommand(NewCmdCreateTracker(commonOpts))
	cmd.AddCommand(NewCmdCreateUser(commonOpts))
	cmd.AddCommand(NewCmdCreateVault(commonOpts))

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
