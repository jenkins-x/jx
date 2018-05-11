package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/issues"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetTrackerOptions the command line options
type GetTrackerOptions struct {
	GetOptions

	Kind string
	Dir  string
}

var (
	getTrackerLong = templates.LongDesc(`
		Display the issue tracker server URLs.

`)

	getTrackerExample = templates.Examples(`
		# List all registered issue tracker server URLs
		jx get tracker
	`)
)

// NewCmdGetTracker creates the command
func NewCmdGetTracker(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetTrackerOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "tracker [flags]",
		Short:   "Display the current registered issue tracker service URLs",
		Long:    getTrackerLong,
		Example: getTrackerExample,
		Aliases: []string{"issue-tracker"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "", "Filters the issue trackers by the kinds: "+strings.Join(issues.IssueTrackerKinds, ", "))
	return cmd
}

// Run implements this command
func (o *GetTrackerOptions) Run() error {
	authConfigSvc, err := o.CreateIssueTrackerAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()
	if len(config.Servers) == 0 {
		o.Printf("No issue trackers registered. To register a new issue tracker use: %s\n", util.ColorInfo("jx create tracker server"))
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
