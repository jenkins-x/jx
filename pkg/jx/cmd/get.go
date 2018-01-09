package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GetOptions struct {
	CommonOptions
}

var (
	get_long = templates.LongDesc(`
		Display one or many resources.

		` + valid_resources + `

`)

	get_example = templates.Examples(`
		# List all pipeines
		kubectl get pipeline

		# List all URLs for services in the current namespace
		kubectl get url
	`)
)

// NewCmdGet creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdGet(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	// retrieve a list of handled resources from printer as valid args
	/*
		validArgs, argAliases := []string{}, []string{}
		p, err := f.Printer(nil, kubectl.PrintOptions{
			ColumnLabels: []string{},
		})
		cmdutil.CheckErr(err)
		if p != nil {
			validArgs = p.HandledResources()
			argAliases = kubectl.ResourceAliases(validArgs)
		}
	*/

	cmd := &cobra.Command{
		Use:     "get TYPE [flags]",
		Short:   "Display one or many resources",
		Long:    get_long,
		Example: get_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunGet(f, out, errOut, cmd, args, options)
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
		/*
			ValidArgs:  validArgs,
			ArgAliases: argAliases,
		*/
	}
	//cmdutil.AddPrinterFlags(cmd)
	/*
		cmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on")
		cmd.Flags().BoolP("watch", "w", false, "After listing/getting the requested object, watch for changes.")
		cmd.Flags().Bool("watch-only", false, "Watch for changes to the requested object(s), without listing/getting first.")
		cmd.Flags().Bool("show-kind", false, "If present, list the resource type for the requested object(s).")
		cmd.Flags().Bool("all-namespaces", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
		cmd.Flags().StringSliceP("label-columns", "L", []string{}, "Accepts a comma separated list of labels that are going to be presented as columns. Names are case-sensitive. You can also use multiple flag options like -L label1 -L label2...")
		cmd.Flags().Bool("export", false, "If true, use 'export' for the resources.  Exported resources are stripped of cluster-specific information.")
	*/
	/*
		usage := "identifying the resource to get from a server."
		cmdutil.AddFilenameOptionFlags(cmd, &options.FilenameOptions, usage)
		cmdutil.AddInclude3rdPartyFlags(cmd)
		cmd.Flags().StringVar(&options.Raw, "raw", options.Raw, "Raw URI to request from the server.  Uses the transport specified by the kubeconfig file.")
	*/
	return cmd
}

// RunGet implements the generic Get command
// TODO: convert all direct flag accessors to a struct and pass that instead of cmd
func RunGet(f cmdutil.Factory, out, errOut io.Writer, cmd *cobra.Command, args []string, options *GetOptions) error {
	if len(args) == 0 {
		fmt.Fprint(errOut, "You must specify the type of resource to get. ", valid_resources)

		usageString := "Required resource not specified."
		return cmdutil.UsageError(cmd, usageString)
	}
	kind := args[0]
	switch kind {
	case "pipeline":
		return options.getPipelines()
	case "pipelines":
		return options.getPipelines()
	case "pipe":
		return options.getPipelines()
	case "pipes":
		return options.getPipelines()

	case "url":
		return options.getURLs()

	default:
		return cmdutil.UsageError(cmd, "Unknown resource kind: %s", kind)
	}
	return nil

}

func (o *GetOptions) getPipelines() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
	if err != nil {
		return err
	}
	jobs, err := jenkins.GetJobs()
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return outputEmptyListWarning(o.Out)
	}

	table := o.CreateTable()
	table.AddRow("Name", "URL")

	for _, job := range jobs {
		table.AddRow(job.Name, job.Url)
	}
	table.Render()
	return nil
}

func (o *GetOptions) getURLs() error {
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

// outputEmptyListWarning outputs a warning indicating that no items are available to display
func outputEmptyListWarning(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s\n", "No resources found.")
	return err
}
