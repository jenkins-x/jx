package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type CreateOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	create_resources = `Valid resource types include:

    * spring (aka 'springboot')
	* cluster
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

	cmd.AddCommand(NewCmdCreateArchetype(f, out, errOut))
	cmd.AddCommand(NewCmdCreateEnv(f, out, errOut))
	cmd.AddCommand(NewCmdCreateSpring(f, out, errOut))
	cmd.AddCommand(NewCmdCreateCluster(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateOptions) Run() error {
	return o.Cmd.Help()
}

// DoImport imports the project created at the given directory
func (o *CreateOptions) DoImport(outDir string) error {
	if o.DisableImport {
		return nil
	}

	importOptions := &ImportOptions{
		CommonOptions:       o.CommonOptions,
		Dir:                 outDir,
		DisableDotGitSearch: true,
	}
	return importOptions.Run()
}

func addCreateAppFlags(cmd *cobra.Command, options *CreateOptions) {
	cmd.Flags().BoolVarP(&options.DisableImport, "no-import", "", false, "Disable import after the creation")
	cmd.Flags().StringVarP(&options.OutDir, "output-dir", "o", "", "Directory to output the project to. Defaults to the current directory")
}
