package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GetOptions struct {
	CommonOptions

	Output string
}

var (
	get_long = templates.LongDesc(`
		Display one or more resources.

		` + valid_resources + `

`)

	get_example = templates.Examples(`
		# List all pipelines
		jx get pipeline

		# List all URLs for services in the current namespace
		jx get url
	`)
)

// NewCmdGet creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdGet(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "get TYPE [flags]",
		Short:   "Display one or more resources",
		Long:    get_long,
		Example: get_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdGetActivity(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetAddon(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetApplications(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetAWSInfo(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetBranchPattern(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetBuild(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetBuildPack(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetChat(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetConfig(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetCVE(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetDevPod(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetEks(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetGit(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetHelmBin(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetIssue(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetIssues(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetLimits(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetPipeline(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetPostPreviewJob(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetPreview(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetQuickstartLocation(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetRelease(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetTeam(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetTeamRole(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetToken(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetTracker(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetURL(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetUser(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetWorkflow(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetVault(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetSecret(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetVaultConfig(f, in, out, errOut))
	cmd.AddCommand(NewCmdGetPlugins(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *GetOptions) Run() error {
	return o.Cmd.Help()
}

// outputEmptyListWarning outputs a warning indicating that no items are available to display
func outputEmptyListWarning(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s\n", "No resources found.")
	return err
}

func (o *GetOptions) addGetFlags(cmd *cobra.Command) {
	o.Cmd = cmd
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "The output format such as 'yaml'")
}

// renderResult renders the result in a given output format
func (o *GetOptions) renderResult(value interface{}, format string) error {
	switch format {
	case "json":
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, e := o.Out.Write(data)
		return e
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		_, e := o.Out.Write(data)
		return e
	default:
		return fmt.Errorf("Unsupported output format: %s", format)
	}
}

func formatInt32(n int32) string {
	return util.Int32ToA(n)
}
