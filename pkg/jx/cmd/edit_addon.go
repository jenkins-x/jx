package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/addon"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	optionEnabled = "enabled"
)

var (
	editAddonLong = templates.LongDesc(`
		Edits an addon
`)

	editAddonExample = templates.Examples(`
		# Enables or disbles an addon
		jx edit addon

		# Enables or disables an addon
		jx edit addon -e true gitea
	`)
)

// EditAddonOptions the options for the create spring command
type EditAddonOptions struct {
	EditOptions

	Name    string
	Enabled string

	IssuesAuthConfigSvc auth.AuthConfigService
}

// NewCmdEditAddon creates a command object for the "create" command
func NewCmdEditAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &EditAddonOptions{
		EditOptions: EditOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addon",
		Short:   "Edits the addon configuration",
		Aliases: []string{"addons"},
		Long:    editAddonLong,
		Example: editAddonExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Enabled, optionEnabled, "e", "", "Enables or disables the addon")

	return cmd
}

// Run implements the command
func (o *EditAddonOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Name = args[0]
	}
	var err error
	charts := kube.AddonCharts
	names := util.SortedMapKeys(charts)
	if o.Name == "" {
		o.Name, err = util.PickName(names, "Pick the addon to configure")
		if err != nil {
			return err
		}
		if o.Name == "" {
			return fmt.Errorf("No addon name chosen")
		}
	}

	addonConfig, err := addon.LoadAddonsConfig()
	if err != nil {
		return err
	}

	config := addonConfig.GetOrCreate(o.Name)
	if o.Enabled != "" {
		text := strings.ToLower(o.Enabled)
		value := false
		if text == "true" {
			value = true
		} else if text != "false" {
			return util.InvalidOption(optionEnabled, o.Enabled, []string{"true", "false"})
		}
		config.Enabled = value
	} else {
		config.Enabled = util.Confirm("Enable addon "+o.Name, config.Enabled, "If an addon is enabled it is installed when using 'jx create cluster' or 'jx install'")
	}

	return addonConfig.Save()
}
