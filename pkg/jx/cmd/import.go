package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/git"
)

type ImportOptions struct {
	CommonOptions

}

func NewCmdImport(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ImportOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Imports a local project into Jenkins",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *ImportOptions) Run() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
	if err != nil {
		return err
	}
	jobs, err := jenkins.GetJobs()
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := git.FindGitConfigDir(dir)
	if err != nil {
		return err
	}
	if root == "" {
		return fmt.Errorf("TODO support non-cloned git repos!")
	}
	out := o.Out
	fmt.Fprintf(out, "Root directory of project is %s\n", root)
	fmt.Fprintf(out, "Has %d jobs\n", len(jobs))
	return nil
}
