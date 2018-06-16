package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// GetHelmBinOptions containers the CLI options
type GetHelmBinOptions struct {
	GetOptions
}

const ()

var (
	helmBinsAliases = []string{
		"branch pattern",
	}

	getHelmBinLong = templates.LongDesc(`
		Display the helm binary name used in pipelines.

		This setting lets you switch from the stable release to early access releases (e.g. from helm 2 <-> 3)
`)

	getHelmBinExample = templates.Examples(`
		# List the git branch patterns for the current team
		jx get helmbin
	`)
)

// NewCmdGetHelmBin creates the new command for: jx get env
func NewCmdGetHelmBin(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetHelmBinOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "helmbin",
		Short:   "Display the helm binary name used in the pipelines",
		Aliases: []string{"helm"},
		Long:    getHelmBinLong,
		Example: getHelmBinExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetHelmBinOptions) Run() error {
	helm, err := o.TeamHelmBin()
	if err != nil {
		return err
	}
	o.Printf("You team uses the helm binary: %s\n", util.ColorInfo(helm))
	o.Printf("To change this value use: %s\n", util.ColorInfo("jx edit helmbin helm3"))
	return nil
}
