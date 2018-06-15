package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

type RepoOptions struct {
	CommonOptions

	Dir         string
	OnlyViewURL bool
}

var (
	repoLong = templates.LongDesc(`
		Opens the web page for the current git repository in a browser

		You can use the '--url' argument to just display the URL without opening it`)

	repoExample = templates.Examples(`
		# Open the git repository in a browser
		jx repo 

		# Print the URL of the git repository
		jx repo -u
`)
)

func NewCmdRepo(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &RepoOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repo"},
		Short:   "Opens the web page for the current git repository in a browser",
		Long:    repoLong,
		Example: repoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *RepoOptions) Run() error {
	gitInfo, provider, _, err := o.createGitProvider(o.Dir)
	if err != nil {
		return err
	}
	if provider == nil {
		return fmt.Errorf("No git provider could be found. Are you in a directory containing a `.git/config` file?")
	}

	fullURL := gitInfo.HttpsURL()
	if fullURL == "" {
		return fmt.Errorf("Could not find URL from git repository %s", gitInfo.URL)
	}
	o.Printf("repository: %s\n", util.ColorInfo(fullURL))
	if !o.OnlyViewURL {
		browser.OpenURL(fullURL)
	}
	return nil
}
