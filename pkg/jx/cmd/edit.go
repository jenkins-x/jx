package cmd

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// EditOptions contains the CLI options
type EditOptions struct {
	commoncmd.CommonOptions
}

var (
	exit_long = templates.LongDesc(`
		Edit a resource

`)

	exit_example = templates.Examples(`
		# Lets edit the staging Environment
		jx edit env staging
	`)
)

// NewCmdEdit creates the edit command
func NewCmdEdit(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &EditOptions{
		commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "edit [flags]",
		Short:   "Edit a resource",
		Long:    exit_long,
		Example: exit_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"modify"},
	}

	cmd.AddCommand(NewCmdCreateBranchPattern(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditAddon(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditBuildpack(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditConfig(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditHelmBin(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditStorage(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditUserRole(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditExtensionsRepository(f, in, out, errOut))
	addTeamSettingsCommandsFromTags(cmd, in, out, errOut, options)
	return cmd
}

// Run implements this command
func (o *EditOptions) Run() error {
	return o.Cmd.Help()
}

func addTeamSettingsCommandsFromTags(baseCmd *cobra.Command, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer, options *EditOptions) error {
	teamSettings := &v1.TeamSettings{}
	value := reflect.ValueOf(teamSettings).Elem()
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := t.Field(i)
		tag := structField.Tag
		command, ok := tag.Lookup("command")
		if !ok {
			continue
		}
		commandUsage, ok := tag.Lookup("commandUsage")
		if !ok {
			continue
		}

		cmd := &cobra.Command{
			Use:   command,
			Short: commandUsage,
			Run: func(cmd *cobra.Command, args []string) {
				var value interface{}
				var err error
				if len(args) > 0 {
					if structField.Type.String() == "string" {
						value = args[0]
					} else if structField.Type.String() == "bool" {
						value, err = strconv.ParseBool(args[0])
						CheckErr(err)
					}
				} else if !options.BatchMode {
					var err error
					if structField.Type.String() == "string" {
						value, err = util.PickValue(commandUsage+":", field.String(), true, "", in, out, errOut)
					} else if structField.Type.String() == "bool" {
						value = util.Confirm(commandUsage+":", field.Bool(), "", in, out, errOut)
					}
					CheckErr(err)
				} else {
					fatal(fmt.Sprintf("No value to set %s", command), 1)
				}

				callback := func(env *v1.Environment) error {
					teamSettings := &env.Spec.TeamSettings
					valueField := reflect.ValueOf(teamSettings).Elem().FieldByName(structField.Name)
					switch value.(type) {
					case string:
						valueField.SetString(value.(string))
					case bool:
						valueField.SetBool(value.(bool))
					}
					log.Infof("Setting the team %s to: %s\n", util.ColorInfo(command), util.ColorInfo(value))
					return nil
				}
				CheckErr(options.ModifyDevEnvironment(callback))
			},
		}

		baseCmd.AddCommand(cmd)
	}
	return nil
}
