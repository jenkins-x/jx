package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

type OpenOptions struct {
	ConsoleOptions
}

func NewCmdOpen(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &OpenOptions{
		ConsoleOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open a service in a browser",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *OpenOptions) Run() error {
	if len(o.Args) == 0 {
		getOptions := GetOptions{
			CommonOptions: o.CommonOptions,
		}
		return getOptions.getURLs()
	}
	name := o.Args[0]
	return o.ConsoleOptions.Open(name, name)
}
