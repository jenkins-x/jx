package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/git"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	"io/ioutil"
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
	root, gitConf, err := git.FindGitConfigDir(dir)
	if err != nil {
		return err
	}
	if root == "" {
		return fmt.Errorf("TODO support non-cloned git repos!")
	}
	out := o.Out

	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return fmt.Errorf("Failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return fmt.Errorf("Could not find any git remotes in the local %s so please specify a git repository on the command line\n", gitConf)
	}
	url := getRemoteUrl(cfg, "upstream")
	if url == "" {
		url = getRemoteUrl(cfg, "origin")
		if url == "" {
			if len(remotes) == 1 {
				for _, r := range remotes {
					u := firstRemoteUrl(r)
					if u != "" {
						url = u
						break
					}
				}
			}
		}
	}
	fmt.Fprintf(out, "Git remote URL: %s\n", url)
	fmt.Fprintf(out, "Has %d jobs\n", len(jobs))
	return nil
}
func firstRemoteUrl(remote *gitcfg.RemoteConfig) string {
	if remote != nil {
		urls := remote.URLs
		if urls != nil && len(urls) > 0 {
			return urls[0]
		}
	}
	return ""
}
func getRemoteUrl(config *gitcfg.Config, name string) string {
	if config.Remotes != nil {
		return firstRemoteUrl(config.Remotes[name])
	}
	return ""
}
