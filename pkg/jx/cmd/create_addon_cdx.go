package cmd

import (
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

const (
	defaultCdxReleaseName = "cdx"
)

var (
	create_addon_cdx_long = templates.LongDesc(`
		Creates the CDX addon

		CDX provides unified Continuous Delivery Environment console to make it easier to do CI / CD and Environments across a number of microservices and teams
`)

	create_addon_cdx_example = templates.Examples(`
		# Create the cdx addon 
		jx create addon cdx
	`)
)

// CreateAddonCDXOptions the options for the create spring command
type CreateAddonCDXOptions struct {
	CreateAddonOptions
}

// NewCmdCreateAddonCDX creates a command object for the "create" command
func NewCmdCreateAddonCDX(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonCDXOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cdx",
		Short:   "Create the CDX addon (a web console for working with CI / CD and Environments)",
		Aliases: []string{"env"},
		Long:    create_addon_cdx_long,
		Example: create_addon_cdx_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, "", defaultCdxReleaseName)
	return cmd
}

// Run implements the command
func (o *CreateAddonCDXOptions) Run() error {
	return o.CreateAddon("cdx")
}
