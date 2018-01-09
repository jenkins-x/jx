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
	"github.com/jenkins-x/golang-jenkins"
)

type ImportOptions struct {
	CommonOptions

	Dir string
	Organisation string
	Repository string


	Jenkins *gojenkins.Jenkins
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
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The source directory to import. If not specified and no arguments supplied then assumes the current directory")
	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Organisation, "name", "n", "", "Specify the git repository name to import the project into (if it is not already in one)")
	return cmd
}

func (o *ImportOptions) Run() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
	if err != nil {
		return err
	}
	o.Jenkins = jenkins

	args := o.Args
	if len(args) == 0 {
		if o.Dir != "" {
			return o.ImportDirectory(o.Dir)
		}
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		return o.ImportDirectory(dir)
	}
	for _, arg := range args {
		err = o.Import(arg)
		if err != nil {
			return fmt.Errorf("Failed to import %s due to %s", arg, err)
		}
	}
	return nil
}


// ImportDirectory finds the git url by looking in the given directory
// and looking for a .git/config file
func (o *ImportOptions) ImportDirectory(dir string) error {
	root, gitConf, err := git.FindGitConfigDir(dir)
	if err != nil {
		return err
	}
	if root == "" {
		return fmt.Errorf("TODO support non-cloned git repos!")
	}
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
	if url != "" {
		return o.Import(url)
	}
	return fmt.Errorf("Could not detect the git URL to import. Please try run this command again and specify a URL")
}

func (o *ImportOptions) Import(url string) error {
	out := o.Out
	jenkins := o.Jenkins
	gitInfo, err := git.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", url, err)
	}
	fmt.Fprintf(out, "Organisation %s and Repository %s\n", gitInfo.Organisation, gitInfo.Name)
	jobs, err := jenkins.GetJobs()
	if err != nil {
		return err
	}
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
