package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetURLOptions the command line options
type GetURLOptions struct {
	GetOptions
}

var (
	get_url_long = templates.LongDesc(`
		Display one or many URLs from the running services.

`)

	get_url_example = templates.Examples(`
		# List all URLs in this namespace
		jx get url
	`)
)

// NewCmdGetURL creates the command
func NewCmdGetURL(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetURLOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "url [flags]",
		Short:   "Display one or many URLs",
		Long:    get_url_long,
		Example: get_url_example,
		Aliases: []string{ "urls"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *GetURLOptions) Run() error {
	f := o.Factory
	client, ns, err := f.CreateClient()
	if err != nil {
		return err
	}
	urls, err := kube.FindServiceURLs(client, ns)
	if err != nil {
		return err
	}
	table := o.CreateTable()
	table.AddRow("Name", "URL")

	for _, url := range urls {
		table.AddRow(url.Name, url.URL)
	}
	table.Render()
	return nil
}
