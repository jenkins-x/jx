package cmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/syntax/syntax.jenkins.io/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	chatKind  = "chat"
	issueKind = "issues"
	wikiKind  = "wiki"
)

var (
	editConfigLong = templates.LongDesc(`
		Edits the project configuration
`)

	editConfigExample = templates.Examples(`
		# Edit the project configuration for the current directory
		jx edit config
	`)

	configKinds = []string{
		chatKind,
		issueKind,
		wikiKind,
	}
)

// EditConfigOptions the options for the create spring command
type EditConfigOptions struct {
	EditOptions

	Dir  string
	Kind string

	IssuesAuthConfigSvc auth.ConfigService
	ChatAuthConfigSvc   auth.ConfigService
}

// NewCmdEditConfig creates a command object for the "create" command
func NewCmdEditConfig(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &EditConfigOptions{
		EditOptions: EditOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Edits the project configuration",
		Aliases: []string{"project"},
		Long:    editConfigLong,
		Example: editConfigExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The root project directory. Defaults to the current dir")
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "", "The kind of configuration to edit root project directory. Possible values "+strings.Join(configKinds, ", "))

	return cmd
}

// Run implements the command
func (o *EditConfigOptions) Run() error {
	pc, fileName, err := v1alpha1.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}
	o.IssuesAuthConfigSvc, err = o.CreateIssueTrackerAuthConfigService()
	if err != nil {
		return err
	}
	o.ChatAuthConfigSvc, err = o.CreateChatAuthConfigService()
	if err != nil {
		return err
	}

	kind := o.Kind
	if kind == "" && !o.BatchMode {
		kind, err = util.PickRequiredNameWithDefault(configKinds, "Which configuration do you want to edit", issueKind, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if kind == "" {
		return fmt.Errorf("No kind option!")
	}
	if util.StringArrayIndex(configKinds, kind) < 0 {
		return util.InvalidOption("kind", kind, configKinds)
	}
	modified := false
	switch kind {
	case chatKind:
		modified, err = o.EditChat(pc)
	case issueKind:
		modified, err = o.EditIssueTracker(pc)
	default:
		return fmt.Errorf("Editing %s is not yet supported!", kind)
	}
	if err != nil {
		return err
	}
	if modified {
		err = pc.SaveConfig(fileName)
		if err != nil {
			return err
		}
		log.Infof("Saved project configuration %s\n", util.ColorInfo(fileName))
	}
	return nil
}

// EditIssueTracker prompts the user to make changes to the issue tracker configuration for the project
func (o *EditConfigOptions) EditIssueTracker(pc *v1alpha1.ProjectConfig) (bool, error) {
	answer := false
	if pc.IssueTracker == nil {
		pc.IssueTracker = &v1alpha1.IssueTrackerConfig{}
		answer = true
	}
	it := pc.IssueTracker

	config := o.IssuesAuthConfigSvc.Config()
	if len(config.Servers) == 0 {
		return answer, fmt.Errorf("No issue tracker servers available. Please add one via: jx create tracker server")
	}
	server, err := config.PickServer("Issue tracker service", o.BatchMode, o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	if server == nil || server.URL == "" {
		return answer, fmt.Errorf("No issue tracker server URL found!")
	}
	it.URL = server.URL
	if server.Kind != "" {
		it.Kind = server.Kind
	}
	answer = true

	it.Project, err = util.PickValue("Issue tracker project name: ", it.Project, true, "", o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	return answer, nil
}

// EditChat prompts the user to make changes to the chat configuration for the project.
func (o *EditConfigOptions) EditChat(pc *v1alpha1.ProjectConfig) (bool, error) {
	answer := false
	if pc.Chat == nil {
		pc.Chat = &v1alpha1.ChatConfig{}
		answer = true
	}
	it := pc.Chat

	config := o.ChatAuthConfigSvc.Config()
	if len(config.Servers) == 0 {
		return answer, fmt.Errorf("No chat servers available. Please add one via: jx create chat server")
	}
	server, err := config.PickServer("Chat service", o.BatchMode, o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	if server == nil || server.URL == "" {
		return answer, fmt.Errorf("No chat server URL found!")
	}
	it.URL = server.URL
	if server.Kind != "" {
		it.Kind = server.Kind
	}
	answer = true

	it.DeveloperChannel, err = util.PickValue("Developer channel: ", it.DeveloperChannel, false, "", o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	it.UserChannel, err = util.PickValue("User channel: ", it.UserChannel, false, "", o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	return answer, nil
}
