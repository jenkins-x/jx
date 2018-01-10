package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jenkins"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	neturl "net/url"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/gits"
	"path/filepath"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	maximumNewDirectoryAttempts = 1000

	DefaultWritePermissions = 0760

	defaultGitIgnoreFile = `
.project
.classpath
.idea
.cache
.DS_Store
*.im?
target
work
`
)

type ImportOptions struct {
	CommonOptions

	RepoURL string

	Dir          string
	Organisation string
	Repository   string
	Credentials  string

	Jenkins    *gojenkins.Jenkins
	GitConfDir string
}

var (
	import_long = templates.LongDesc(`
		Imports a git repository or folder into Jenkins X.

		If you specify no other options or arguments then the current directory is imported.
	    Or you can use '--dir' to specify a directory to import.

	    You can specify the git URL as an argument.`)

	import_example = templates.Examples(`
		# Import the current folder
		jx import

		# Import a different folder
		jx import /foo/bar

		# Import a git repository from a URL
		jx import -repo https://github.com/jenkins-x/spring-boot-web-example.git`)
)

func NewCmdImport(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ImportOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a local project into Jenkins",
		Long:    import_long,
		Example: import_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.RepoURL, "url", "u", "", "The git clone URL to clone into the current directory and then import")
	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Organisation, "name", "n", "", "Specify the git repository name to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Credentials, "credentials", "c", "jenkins-x-github", "The Jenkins credentials name used by the job")
	return cmd
}

func (o *ImportOptions) Run() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
	if err != nil {
		return err
	}
	o.Jenkins = jenkins

	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Dir = dir
	}
	args := o.Args
	if len(args) > 0 {
		o.Dir = args[0]
	}

	if o.RepoURL != "" {
		err = o.CloneRepository()
		if err != nil {
			return err
		}
	} else {
		err = o.DiscoverGit()
		if err != nil {
			return err
		}

		if o.RepoURL == "" {
			err = o.DiscoverRemoteGitURL()
			if err != nil {
				return err
			}
		}
	}
	for _, arg := range args {
		err = o.Import(arg)
		if err != nil {
			return fmt.Errorf("Failed to import %s due to %s", arg, err)
		}
	}
	return nil
}

func (o *ImportOptions) CloneRepository() error {
	url := o.RepoURL
	if url == "" {
		return fmt.Errorf("No git repository URL defined!")
	}
	gitInfo, err := gits.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", url, err)
	}
	cloneDir, err := util.CreateUniqueDirectory(o.Dir, gitInfo.Name, maximumNewDirectoryAttempts)
	if err != nil {
		return err
	}
	err = gits.GitClone(url, cloneDir)
	if err != nil {
		return err
	}
	o.Dir = cloneDir
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DiscoverGit() error {
	root, gitConf, err := gits.FindGitConfigDir(o.Dir)
	if err != nil {
		return err
	}
	if root != "" {
		o.Dir = root
		o.GitConfDir = gitConf
		return nil
	}

	dir := o.Dir
	if dir == "" {
		return fmt.Errorf("No directory specified!")
	}

	// lets prompt the user to initiialse the git repository
	o.Printf("The directory %s is not yet using git\n", dir)
	flag := false
	prompt := &survey.Confirm{
	    Message: "Would you like to initialise git now?",
	}
	err = survey.AskOne(prompt, &flag, nil)
	if err != nil {
	  return err
	}
	if !flag {
		return fmt.Errorf("Please initialise git yourself then try again")
	}

	err = gits.GitInit(dir)
	if err != nil {
	  return err
	}
	o.GitConfDir = filepath.Join(dir, ".git.conf")
	err = o.DefaultGitIgnore()
	if err != nil {
	  return err
	}
	err = gits.GitAdd(dir, ".gitignore")
	if err != nil {
	  return err
	}
	err = gits.GitAdd(dir, "*")
	if err != nil {
	  return err
	}

	err = gits.GitStatus(dir)
	if err != nil {
	  return err
	}

	message := ""
	messagePrompt := &survey.Input{
	    Message: "Commit message: ",
	    Default: "Initial import",
	}
	err = survey.AskOne(messagePrompt, &message, nil)
	if err != nil {
	  return err
	}
	return gits.GitCommit(dir, message)
}


// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DefaultGitIgnore() error {
	name := filepath.Join(o.Dir, ".gitignore")
	exists, err := util.FileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		data := []byte(defaultGitIgnoreFile)
		err = ioutil.WriteFile(name, data, DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to write %s due to %s", name, err)
		}
	}
	return nil
}


// DiscoverRemoteGitURL finds the git url by looking in the directory
// and looking for a .git/config file
func (o *ImportOptions) DiscoverRemoteGitURL() error {
	gitConf := o.GitConfDir
	if gitConf == "" {
		return fmt.Errorf("No GitConfDir defined!")
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
		o.RepoURL = url
	}
	return nil
}

func (o *ImportOptions) Import(url string) error {
	out := o.Out
	jenk := o.Jenkins
	gitInfo, err := gits.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("Failed to parse git URL %s due to: %s", url, err)
	}
	org := gitInfo.Organisation
	folder, err := jenk.GetJob(org)
	if err != nil {
		// could not find folder so lets try create it
		jobUrl := util.UrlJoin(jenk.BaseURL(), jenk.GetJobURLPath(org))
		folderXml := jenkins.CreateFolderXml(jobUrl, org)
		//fmt.Fprintf(out, "XML: %s\n", folderXml)
		err = jenk.CreateJobWithXML(folderXml, org)
		if err != nil {
			return fmt.Errorf("Failed to create the %s folder in jenkins: %s", org, err)
		}
		//fmt.Fprintf(out, "Created Jenkins folder: %s\n", org)
	} else {
		c := folder.Class
		if c != "com.cloudbees.hudson.plugins.folder.Folder" {
			fmt.Fprintf(out, "Warning the folder %s is of class %s", org, c)
		}
	}
	projectXml := jenkins.CreateMultiBranchProjectXml(gitInfo, o.Credentials)
	jobName := gitInfo.Name
	job, err := jenk.GetJobByPath(org, jobName)
	if err == nil {
		return fmt.Errorf("Job already exists in Jenkins at " + job.Url)
	}
	//fmt.Fprintf(out, "Creating MultiBranchProject %s from XML: %s\n", jobName, projectXml)
	err = jenk.CreateFolderJobWithXML(projectXml, org, jobName)
	if err != nil {
		return fmt.Errorf("Failed to create MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
	}
	job, err = jenk.GetJobByPath(org, jobName)
	if err != nil {
		return fmt.Errorf("Failed to find the MultiBranchProject job %s in folder %s due to: %s", jobName, org, err)
	}
	fmt.Fprintf(out, "Created Project: %s\n", job.Url)
	params := neturl.Values{}
	err = jenk.Build(job, params)
	if err != nil {
		return fmt.Errorf("Failed to trigger job %s due to %s", job.Url, err)
	}
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
