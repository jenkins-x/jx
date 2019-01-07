package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteTrackerServer_long = templates.LongDesc(`
		Deletes one or more issue tracker servers from your local settings
`)

	deleteTrackerServer_example = templates.Examples(`
		# Deletes an issue tracker server
		jx delete tracker server MyProvider
	`)
)

// DeleteTrackerServerOptions the options for the create spring command
type DeleteTrackerServerOptions struct {
	CommonOptions

	IgnoreMissingServer bool
}

// NewCmdDeleteTrackerServer defines the command
func NewCmdDeleteTrackerServer(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteTrackerServerOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Deletes one or more issue tracker server(s)",
		Long:    deleteTrackerServer_long,
		Example: deleteTrackerServer_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.IgnoreMissingServer, "ignore-missing", "i", false, "Silently ignore attempts to remove an issue tracker server name that does not exist")
	return cmd
}

// Run implements the command
func (o *DeleteTrackerServerOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing issue tracker server name argument")
	}
	authConfigSvc, err := o.createIssueTrackerAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	serverNames := config.GetServerNames()
	for _, arg := range args {
		idx := config.IndexOfServerName(arg)
		if idx < 0 {
			if o.IgnoreMissingServer {
				return nil
			}
			return util.InvalidArg(arg, serverNames)
		}
		config.Servers = append(config.Servers[0:idx], config.Servers[idx+1:]...)
	}
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	log.Infof("Deleted issue tracker servers: %s from local settings\n", util.ColorInfo(strings.Join(args, ", ")))
	return nil
}
