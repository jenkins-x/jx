package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/chats"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetChatOptions the command line options
type GetChatOptions struct {
	GetOptions

	Kind string
	Dir  string
}

var (
	getChatLong = templates.LongDesc(`
		Display the chat server URLs.

`)

	getChatExample = templates.Examples(`
		# List all registered chat server URLs
		jx get chat
	`)
)

// NewCmdGetChat creates the command
func NewCmdGetChat(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetChatOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chat [flags]",
		Short:   "Display the current registered chat service URLs",
		Long:    getChatLong,
		Example: getChatExample,
		Aliases: []string{"slack"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "", "Filters the chats by the kinds: "+strings.Join(chats.ChatKinds, ", "))
	return cmd
}

// Run implements this command
func (o *GetChatOptions) Run() error {
	authConfigSvc, err := o.CreateChatAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	if len(config.Servers) == 0 {
		o.Printf("No chat servers registered. To register a new chat servers use: %s\n", util.ColorInfo("jx create chat server"))
		return nil
	}
	filterKind := o.Kind

	table := o.CreateTable()
	if filterKind == "" {
		table.AddRow("Name", "Kind", "URL")
	} else {
		table.AddRow(strings.ToUpper(filterKind), "URL")
	}

	for _, s := range config.Servers {
		kind := s.Kind
		if filterKind == "" || filterKind == kind {
			table.AddRow(s.Name, kind, s.URL)
		} else if filterKind == kind {
			table.AddRow(s.Name, s.URL)
		}
	}
	table.Render()
	return nil
}
