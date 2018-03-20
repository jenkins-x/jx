package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/jenkins-x/draft-repo/pkg/draft/pack"
	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	jxdraft "github.com/jenkins-x/jx/pkg/draft"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//_ "github.com/Azure/draft/pkg/linguist"
)

const (
	PlaceHolderAppName     = "REPLACE_ME_APP_NAME"
	PlaceHolderGitProvider = "REPLACE_ME_GIT_PROVIDER"
	PlaceHolderOrg         = "REPLACE_ME_ORG"

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

	Dir                     string
	Organisation            string
	Repository              string
	Credentials             string
	AppName                 string
	GitHub                  bool
	DryRun                  bool
	SelectAll               bool
	DisableDraft            bool
	DisableJenkinsfileCheck bool
	SelectFilter            string
	Jenkinsfile             string
	BranchPattern           string
	GitRepositoryOptions    gits.GitRepositoryOptions
	ImportGitCommitMessage  string

	DisableDotGitSearch bool
	Jenkins             *gojenkins.Jenkins
	GitConfDir          string
	GitProvider         gits.GitProvider
}

var (
	import_long = templates.LongDesc(`
		Imports a local folder or git repository into Jenkins X.

		If you specify no other options or arguments then the current directory is imported.
	    Or you can use '--dir' to specify a directory to import.

	    You can specify the git URL as an argument.
	    
		For more documentation see: [http://jenkins-x.io/developing/import/](http://jenkins-x.io/developing/import/)
	    
	`)

	import_example = templates.Examples(`
		# Import the current folder
		jx import

		# Import a different folder
		jx import /foo/bar

		# Import a git repository from a URL
		jx import --url https://github.com/jenkins-x/spring-boot-web-example.git

        # Select a number of repositories from a github organisation
		jx import --github --org myname 

        # Import all repositories from a github organisation selecting ones to not import
		jx import --github --org myname --all 

        # Import all repositories from a github organisation which contain the text foo
		jx import --github --org myname --all --filter foo 
		`)
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
		Short:   "Imports a local project or git repository into Jenkins",
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
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wish to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "", false, "If selecting projects to import from a git provider this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "", "", "If selecting projects to import from a git provider this filters the list of repositories")

	options.addImportFlags(cmd, false)

	return cmd
}

func (options *ImportOptions) addImportFlags(cmd *cobra.Command, createProject bool) {
	notCreateProject := func(text string) string {
		if createProject {
			return ""
		}
		return text
	}
	cmd.Flags().StringVarP(&options.Organisation, "org", "", "", "Specify the git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Repository, "name", "", notCreateProject("n"), "Specify the git repository name to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Credentials, "credentials", notCreateProject("c"), "", "The Jenkins credentials name used by the job")
	cmd.Flags().StringVarP(&options.Jenkinsfile, "jenkinsfile", notCreateProject("j"), "", "The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used")
	cmd.Flags().BoolVarP(&options.DryRun, "dry-run", "", false, "Performs local changes to the repo but skips the import into Jenkins X")
	cmd.Flags().BoolVarP(&options.DisableDraft, "no-draft", "", false, "Disable Draft from trying to default a Dockerfile and Helm Chart")
	cmd.Flags().BoolVarP(&options.DisableJenkinsfileCheck, "no-jenkinsfile", "", false, "Disable defaulting a Jenkinsfile if its missing")
	cmd.Flags().StringVarP(&options.ImportGitCommitMessage, "import-commit-message", "", "", "The git commit message for the import")
	cmd.Flags().StringVarP(&options.BranchPattern, "branches", "", "", "The branch pattern for branches to trigger CI / CD pipelines on. Defaults to '"+jenkins.DefaultBranchPattern+"'")

	options.addCommonFlags(cmd)
	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)
}

func (o *ImportOptions) Run() error {
	f := o.Factory
	f.SetBatch(o.BatchMode)

	jenkins, err := f.CreateJenkinsClient()
	if err != nil {
		return err
	}

	o.Jenkins = jenkins

	client, ns, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	o.currentNamespace = ns
	o.kubeClient = client

	var userAuth *auth.UserAuth
	if o.GitProvider == nil {
		authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
		if err != nil {
			return err
		}
		config := authConfigSvc.Config()
		server := config.GetOrCreateServer(gits.GitHubHost)
		userAuth, err = config.PickServerUserAuth(server, "git user name", o.BatchMode)
		if err != nil {
			return err
		}
		o.GitProvider, err = gits.CreateProvider(server, userAuth)
		if err != nil {
			return err
		}
	}
	if o.Organisation == "" {
		o.Organisation = userAuth.Username
	}

	if o.GitHub {
		return o.ImportProjectsFromGitHub()
	}
	if o.Dir == "" {
		args := o.Args
		if len(args) > 0 {
			o.Dir = args[0]
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			o.Dir = dir
		}
	}
	_, o.AppName = filepath.Split(o.Dir)

	checkForJenkinsfile := o.Jenkinsfile == "" && !o.DisableJenkinsfileCheck
	shouldClone := checkForJenkinsfile || !o.DisableDraft

	if o.RepoURL != "" {
		if shouldClone {
			// lets make sure there's a .git at the end for github URLs
			err = o.CloneRepository()
			if err != nil {
				return err
			}
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

	if !o.DisableDraft {
		err = o.DraftCreate()
		if err != nil {
			return err
		}

	}

	if o.RepoURL == "" {
		err = o.CreateNewRemoteRepository()
		if err != nil {
			return err
		}
	} else {
		if shouldClone {
			err = gits.GitPush(o.Dir)
			if err != nil {
				return err
			}
		}
	}

	err = o.checkChartmuseumCredentialExists()
	if err != nil {
		return err
	}

	if o.DryRun {
		log.Infof("dry-run so skipping import to Jenkins X")
		return nil
	}

	return o.DoImport()
}

func (o *ImportOptions) ImportProjectsFromGitHub() error {

	repos, err := gits.PickRepositories(o.GitProvider, o.Organisation, "Which repositories do you want to import", o.SelectAll, o.SelectFilter)
	if err != nil {
		return err
	}

	o.Printf("Selected repositories\n")
	for _, r := range repos {
		o2 := ImportOptions{
			CommonOptions:           o.CommonOptions,
			Dir:                     o.Dir,
			RepoURL:                 r.CloneURL,
			Organisation:            o.Organisation,
			Repository:              r.Name,
			Jenkins:                 o.Jenkins,
			GitProvider:             o.GitProvider,
			DisableJenkinsfileCheck: o.DisableJenkinsfileCheck,
			DisableDraft:            o.DisableDraft,
		}
		o.Printf("Importing repository %s\n", util.ColorInfo(r.Name))
		err = o2.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ImportOptions) DraftCreate() error {

	draftDir, err := util.DraftDir()
	if err != nil {
		return err
	}
	draftHome := draftpath.Home(draftDir)

	// lets make sure we have the latest draft packs
	initOpts := InitOptions{
		CommonOptions: CommonOptions{
			Out: o.Out,
		},
	}
	err = initOpts.initDraft()
	if err != nil {
		return err
	}

	// TODO this is a workaround of this draft issue:
	// https://github.com/Azure/draft/issues/476
	dir := o.Dir
	pomName := filepath.Join(dir, "pom.xml")
	exists, err := util.FileExists(pomName)
	if err != nil {
		return err
	}
	lpack := ""
	if exists {
		lpack = filepath.Join(draftHome.Packs(), "github.com/jenkins-x/draft-packs/packs/java")
	} else {
		// pack detection time
		lpack, err = jxdraft.DoPackDetection(draftHome, o.Out, dir)

		if err != nil {
			return err
		}
	}
	chartsDir := filepath.Join(dir, "charts")
	exists, err = util.FileExists(chartsDir)
	if exists {
		log.Warn("existing charts folder found so skipping 'draft create' step\n")
		return nil
	}

	err = pack.CreateFrom(dir, lpack)
	if err != nil {
		return err
	}
	if err != nil {
		// lets ignore draft errors as sometimes it can't find a pack - e.g. for environments
		o.Printf(util.ColorWarning("WARNING: Failed to run draft create in %s due to %s"), dir, err)
		//return fmt.Errorf("Failed to run draft create in %s due to %s", dir, err)
	}

	// lets rename the chart to be the same as our app name
	err = o.renameChartToMatchAppName()
	if err != nil {
		return err
	}

	//walk through every file in the given dir and update the placeholders
	err = o.replacePlaceholders()
	if err != nil {
		return err
	}

	err = gits.GitAdd(dir, "*")
	if err != nil {
		return err
	}
	err = gits.GitCommitIfChanges(dir, "Draft create")
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) CreateNewRemoteRepository() error {
	f := o.Factory
	authConfigSvc, err := f.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	dir := o.Dir
	_, defaultRepoName := filepath.Split(dir)

	details, err := gits.PickNewGitRepository(o.Out, o.BatchMode, authConfigSvc, defaultRepoName, o.GitRepositoryOptions)
	if err != nil {
		return err
	}
	repo, err := details.CreateRepository()
	if err != nil {
		return err
	}
	o.GitProvider = details.GitProvider

	o.RepoURL = repo.CloneURL
	pushGitURL, err := gits.GitCreatePushURL(repo.CloneURL, details.User)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "remote", "add", "origin", pushGitURL)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "push", "-u", "origin", "master")
	if err != nil {
		return err
	}
	o.Printf("Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	return nil
}

func (o *ImportOptions) CloneRepository() error {
	url := o.RepoURL
	if url == "" {
		return fmt.Errorf("no git repository URL defined!")
	}
	gitInfo, err := gits.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse git URL %s due to: %s", url, err)
	}
	if gitInfo.Host == gits.GitHubHost && strings.HasPrefix(gitInfo.Scheme, "http") {
		if !strings.HasSuffix(url, ".git") {
			url += ".git"
		}
		o.RepoURL = url
	}
	cloneDir, err := util.CreateUniqueDirectory(o.Dir, gitInfo.Name, util.MaximumNewDirectoryAttempts)
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
	if !o.DisableDotGitSearch {
		root, gitConf, err := gits.FindGitConfigDir(o.Dir)
		if err != nil {
			return err
		}
		if root != "" {
			o.Dir = root
			o.GitConfDir = gitConf
			return nil
		}
	}

	dir := o.Dir
	if dir == "" {
		return fmt.Errorf("no directory specified!")
	}

	// lets prompt the user to initialise the git repository
	if !o.BatchMode {
		o.Printf("The directory %s is not yet using git\n", util.ColorInfo(dir))
		flag := false
		prompt := &survey.Confirm{
			Message: "Would you like to initialise git now?",
			Default: true,
		}
		err := survey.AskOne(prompt, &flag, nil)
		if err != nil {
			return err
		}
		if !flag {
			return fmt.Errorf("please initialise git yourself then try again")
		}
	}
	err := gits.GitInit(dir)
	if err != nil {
		return err
	}
	o.GitConfDir = filepath.Join(dir, ".git/config")
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

	message := o.ImportGitCommitMessage
	if message == "" {
		if o.BatchMode {
			message = "Initial import"
		} else {
			messagePrompt := &survey.Input{
				Message: "Commit message: ",
				Default: "Initial import",
			}
			err = survey.AskOne(messagePrompt, &message, nil)
			if err != nil {
				return err
			}
		}
	}
	err = gits.GitCommitIfChanges(dir, message)
	if err != nil {
		return err
	}
	o.Printf("\nGit repository created\n")
	return nil
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
			return fmt.Errorf("failed to write %s due to %s", name, err)
		}
	}
	return nil
}

// DiscoverRemoteGitURL finds the git url by looking in the directory
// and looking for a .git/config file
func (o *ImportOptions) DiscoverRemoteGitURL() error {
	gitConf := o.GitConfDir
	if gitConf == "" {
		return fmt.Errorf("no GitConfDir defined!")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return fmt.Errorf("failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return nil
	}
	url := gits.GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = gits.GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.pickRemoteURL(cfg)
			if err != nil {
				return err
			}
		}
	}
	if url != "" {
		o.RepoURL = url
	}
	return nil
}

func (o *ImportOptions) DoImport() error {
	if o.Jenkins == nil {
		jclient, err := o.Factory.CreateJenkinsClient()
		if err != nil {
			return err
		}
		o.Jenkins = jclient
	}
	gitURL := o.RepoURL
	gitProvider := o.GitProvider
	if gitProvider == nil {
		p, err := o.gitProviderForURL(gitURL, "user name to register webhook")
		if err != nil {
			return err
		}
		gitProvider = p
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	jenkinsfile := o.Jenkinsfile
	if jenkinsfile == "" {
		jenkinsfile = jenkins.DefaultJenkinsfile
	}
	return jenkins.ImportProject(o.Out, o.Jenkins, gitURL, o.Dir, jenkinsfile, o.BranchPattern, o.Credentials, false, gitProvider, authConfigSvc)
}

func (o *ImportOptions) replacePlaceholders() error {
	log.Infof("replacing placeholders in direcory %s\n", o.Dir)
	log.Infof("app name: %s, git server: %s, org: %s\n", o.AppName, o.GitRepositoryOptions.ServerURL, o.Organisation)

	if err := filepath.Walk(o.Dir, func(f string, fi os.FileInfo, err error) error {
		if fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if !fi.IsDir() {
			input, err := ioutil.ReadFile(f)
			if err != nil {
				log.Errorf("failed to read file %s: %v", f, err)
				return err
			}

			lines := strings.Split(string(input), "\n")

			for i, line := range lines {
				line = strings.Replace(line, PlaceHolderAppName, o.AppName, -1)
				line = strings.Replace(line, PlaceHolderGitProvider, o.GitRepositoryOptions.ServerURL, -1)
				line = strings.Replace(line, PlaceHolderOrg, o.Organisation, -1)
				lines[i] = line
			}
			output := strings.Join(lines, "\n")
			err = ioutil.WriteFile(f, []byte(output), 0644)
			if err != nil {
				log.Errorf("failed to write file %s: %v", f, err)
				return err
			}
		}
		return nil

	}); err != nil {
		return fmt.Errorf("error replacing placeholders %v", err)
	}

	return nil
}

func (o *ImportOptions) addAppNameToGeneratedFile(filename, field, value string) error {
	dir := filepath.Join(o.Dir, "charts", o.AppName)
	file := filepath.Join(dir, filename)
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		// no file so lets ignore this
		return nil
	}
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, field) {
			lines[i] = fmt.Sprintf("%s%s", field, value)
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) checkChartmuseumCredentialExists() error {

	name := jenkins.DefaultJenkinsCredentialsPrefix + jenkins.Chartmuseum
	_, err := o.Jenkins.GetCredential(name)

	if err != nil {
		secret, err := o.kubeClient.CoreV1().Secrets(o.currentNamespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			fmt.Errorf("error getting %s secret %v", name, err)
		}

		data := secret.Data
		username := string(data["BASIC_AUTH_USER"])
		password := string(data["BASIC_AUTH_PASS"])

		err = o.Jenkins.CreateCredential(name, username, password)
		if err != nil {
			return fmt.Errorf("error creating jenkins credential %s %v", name, err)
		}
	}
	return nil
}
func (o *ImportOptions) renameChartToMatchAppName() error {
	var oldChartsDir string
	dir := o.Dir
	if err := filepath.Walk(dir, func(f string, fi os.FileInfo, err error) error {
		if fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if fi.IsDir() {
			switch fi.Name() {
			case "csharp":
				oldChartsDir = filepath.Join(dir, "charts", "csharp")
				break
			case "go":
				oldChartsDir = filepath.Join(dir, "charts", "go")
				break
			case "gradle":
				oldChartsDir = filepath.Join(dir, "charts", "gradle")
				break
			case "java":
				oldChartsDir = filepath.Join(dir, "charts", "java")
				break
			case "javascript":
				oldChartsDir = filepath.Join(dir, "charts", "javascript")
				break
			case "php":
				oldChartsDir = filepath.Join(dir, "charts", "php")
				break
			case "python":
				oldChartsDir = filepath.Join(dir, "charts", "python")
				break
			case "ruby":
				oldChartsDir = filepath.Join(dir, "charts", "ruby")
				break
			case "swift":
				oldChartsDir = filepath.Join(dir, "charts", "swift")
				break
			}
		}
		return nil

	}); err != nil {
		return fmt.Errorf("error matching a jenkins x draft pack name with chart folder %v", err)
	}

	if oldChartsDir != "" {
		// chart expects folder name to be the same as app name
		newChartsDir := filepath.Join(dir, "charts", o.AppName)

		exists, err := util.FileExists(oldChartsDir)
		if err != nil {
			return err
		}
		if exists {
			err = util.RenameDir(oldChartsDir, newChartsDir, false)
			if err != nil {
				return fmt.Errorf("error renaming %s to %s, %v", oldChartsDir, newChartsDir, err)
			}
			_, err = os.Stat(newChartsDir)
			if err != nil {
				return err
			}
		}
		// now update the chart.yaml
		err = o.addAppNameToGeneratedFile("Chart.yaml", "name: ", o.AppName)
		if err != nil {
			return err
		}
	}
	return nil
}
