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

	Namespace   string
	Environment string
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
		Use:     "urls [flags]",
		Short:   "Display one or many URLs",
		Long:    get_url_long,
		Example: get_url_example,
		Aliases: []string{"url"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addGetUrlFlags(cmd)
	return cmd
}

func (o *GetURLOptions) addGetUrlFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Specifies the namespace name to look inside")
	cmd.Flags().StringVarP(&o.Environment, "env", "e", "", "Specifies the Environment name to look inside")
}

// Run implements this command
func (o *GetURLOptions) Run() error {
	client, ns, err := o.KubeClient()
	if err != nil {
		return err
	}
	if o.Namespace != "" {
		ns = o.Namespace
	} else if o.Environment != "" {
		ns, err = o.findEnvironmentNamespace(o.Environment)
		if err != nil {
			return err
		}
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
