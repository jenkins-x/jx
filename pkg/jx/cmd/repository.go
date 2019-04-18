package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

type RepoOptions struct {
	*opts.CommonOptions

	Dir         string
	OnlyViewURL bool
}

var (
	repoLong = templates.LongDesc(`
		Opens the web page for the current Git repository in a browser

		You can use the '--url' argument to just display the URL without opening it`)

	repoExample = templates.Examples(`
		# Open the Git repository in a browser
		jx repo 

		# Print the URL of the Git repository
		jx repo -u
`)
)

func NewCmdRepo(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &RepoOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repo"},
		Short:   "Opens the web page for the current Git repository in a browser",
		Long:    repoLong,
		Example: repoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *RepoOptions) Run() error {
	gitInfo, provider, _, err := o.CreateGitProvider(o.Dir)
	if err != nil {
		return err
	}
	if provider == nil {
		return fmt.Errorf("No Git provider could be found. Are you in a directory containing a `.git/config` file?")
	}

	fullURL := gitInfo.HttpsURL()
	if fullURL == "" {
		return fmt.Errorf("Could not find URL from Git repository %s", gitInfo.URL)
	}
	log.Infof("repository: %s\n", util.ColorInfo(fullURL))
	if !o.OnlyViewURL {
		browser.OpenURL(fullURL)
	}
	return nil
}
