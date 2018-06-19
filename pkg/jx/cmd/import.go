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
	"github.com/jenkins-x/jx/pkg/config"
	jxdraft "github.com/jenkins-x/jx/pkg/draft"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
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

	jenkinsfileBackupSuffix = ".backup"

	minimumMavenDeployVersion = "2.8.2"

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

type CallbackFn func() error

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
	ListDraftPacks          bool
	DraftPack               string
	DefaultOwner            string

	DisableDotGitSearch   bool
	InitialisedGit        bool
	Jenkins               *gojenkins.Jenkins
	GitConfDir            string
	GitServer             *auth.AuthServer
	GitUserAuth           *auth.UserAuth
	GitProvider           gits.GitProvider
	PostDraftPackCallback CallbackFn
	DisableMaven          bool
}

var (
	import_long = templates.LongDesc(`
		Imports a local folder or git repository into Jenkins X.

		If you specify no other options or arguments then the current directory is imported.
	    Or you can use '--dir' to specify a directory to import.

	    You can specify the git URL as an argument.
	    
		For more documentation see: [https://jenkins-x.io/developing/import/](https://jenkins-x.io/developing/import/)
	    
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
	cmd.Flags().StringVarP(&options.ImportGitCommitMessage, "import-commit-message", "", "", "Should we override the Jenkinsfile in the project?")
	cmd.Flags().StringVarP(&options.BranchPattern, "branches", "", "", "The branch pattern for branches to trigger CI/CD pipelines on")
	cmd.Flags().BoolVarP(&options.ListDraftPacks, "list-packs", "", false, "list available draft packs")
	cmd.Flags().StringVarP(&options.DraftPack, "pack", "", "", "The name of the pack to use")
	cmd.Flags().StringVarP(&options.DefaultOwner, "default-owner", "", "someone", "The default user/organisation used if no user is found for the current git repository being imported")

	options.addCommonFlags(cmd)
	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)
}

func (o *ImportOptions) Run() error {
	if o.ListDraftPacks {
		packs, err := allDraftPacks()
		if err != nil {
			log.Error(err.Error())
			return err
		}
		o.Printf("Available draft packs:\n")
		for i := 0; i < len(packs); i++ {
			o.Printf(packs[i] + "\n")
		}
		return nil
	}

	o.Factory.SetBatch(o.BatchMode)

	var err error
	if !o.DryRun {
		o.Jenkins, err = o.JenkinsClient()
		if err != nil {
			return err
		}

		_, _, err = o.KubeClient()
		if err != nil {
			return err
		}
	}

	var userAuth *auth.UserAuth
	if o.GitProvider == nil {
		authConfigSvc, err := o.CreateGitAuthConfigServiceDryRun(o.DryRun)
		if err != nil {
			return err
		}
		config := authConfigSvc.Config()
		var server *auth.AuthServer
		if o.RepoURL != "" {
			gitInfo, err := gits.ParseGitURL(o.RepoURL)
			if err != nil {
				return err
			}
			serverUrl := gitInfo.HostURLWithoutUser()
			server = config.GetOrCreateServer(serverUrl)
		} else {
			server, err = config.PickOrCreateServer(gits.GitHubURL, "Which git service do you wish to use", o.BatchMode)
			if err != nil {
				return err
			}
		}
		o.Debugf("Found git server %s %s\n", server.URL, server.Kind)
		userAuth, err = config.PickServerUserAuth(server, "git user name:", o.BatchMode)
		if err != nil {
			return err
		}
		updated := false
		if server.Kind == "" {
			server.Kind, err = o.GitServerHostURLKind(server.URL)
			if err != nil {
				return err
			}
			updated = true
		}
		if userAuth.IsInvalid() {
			f := func(username string) error {
				gits.PrintCreateRepositoryGenerateAccessToken(server, username, o.Out)
				return nil
			}
			err = config.EditUserAuth(server.Label(), userAuth, userAuth.Username, true, o.BatchMode, f)
			if err != nil {
				return err
			}

			o.Credentials, err = o.updatePipelineGitCredentialsSecret(server, userAuth)
			if err != nil {
				return err
			}

			updated = true

			// TODO lets verify the auth works?
			if userAuth.IsInvalid() {
				return fmt.Errorf("You did not properly define the user authentication!")
			}
		}
		if updated {
			err = authConfigSvc.SaveUserAuth(server.URL, userAuth)
			if err != nil {
				return fmt.Errorf("Failed to store git auth configuration %s", err)
			}
		}

		o.GitServer = server
		o.GitUserAuth = userAuth
		o.GitProvider, err = gits.CreateProvider(server, userAuth)
		if err != nil {
			return err
		}
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
	o.AppName = kube.ToValidName(strings.ToLower(o.AppName))

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
	err = o.fixDockerIgnoreFile()
	if err != nil {
		return err
	}

	err = o.fixMaven()
	if err != nil {
		return err
	}

	if o.RepoURL == "" {
		if !o.DryRun {
			err = o.CreateNewRemoteRepository()
			if err != nil {
				return err
			}
		}
	} else {
		if shouldClone {
			err = gits.GitPush(o.Dir)
			if err != nil {
				return err
			}
		}
	}

	if o.DryRun {
		o.Printf("dry-run so skipping import to Jenkins X\n")
		return nil
	}

	err = o.checkChartmuseumCredentialExists()
	if err != nil {
		return err
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
		CommonOptions: o.CommonOptions,
	}
	packsDir, err := initOpts.initBuildPacks()
	if err != nil {
		return err
	}

	// TODO this is a workaround of this draft issue:
	// https://github.com/Azure/draft/issues/476
	dir := o.Dir

	jenkinsfile := filepath.Join(dir, "Jenkinsfile")
	pomName := filepath.Join(dir, "pom.xml")
	gradleName := filepath.Join(dir, "build.gradle")
	lpack := ""
	customDraftPack := o.DraftPack
	if len(customDraftPack) == 0 {
		projectConfig, _, err := config.LoadProjectConfig(dir)
		if err != nil {
			return err
		}
		customDraftPack = projectConfig.BuildPack
	}

	if len(customDraftPack) > 0 {
		log.Info("trying to use draft pack: " + customDraftPack + "\n")
		lpack = filepath.Join(packsDir, customDraftPack)
		f, err := util.FileExists(lpack)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if f == false {
			log.Error("Could not find pack: " + customDraftPack + " going to try detect which pack to use")
			lpack = ""
		}

	}

	if len(lpack) == 0 {
		if exists, err := util.FileExists(pomName); err == nil && exists {
			pack, err := cmdutil.PomFlavour(pomName)
			if err != nil {
				return err
			}
			if len(pack) > 0 {
				if pack == cmdutil.LIBERTY {
					lpack = filepath.Join(packsDir, "liberty")
				} else if pack == cmdutil.APPSERVER {
					lpack = filepath.Join(packsDir, "appserver")
				} else {
					log.Warn("Do not know how to handle pack: " + pack)
				}
			} else {
				lpack = filepath.Join(packsDir, "maven")
			}

			exists, _ = util.FileExists(lpack)
			if !exists {
				log.Warn("defaulting to maven pack")
				lpack = filepath.Join(packsDir, "maven")
			}
		} else if exists, err := util.FileExists(gradleName); err == nil && exists {
			lpack = filepath.Join(packsDir, "gradle")
		} else {
			// pack detection time
			lpack, err = jxdraft.DoPackDetection(draftHome, o.Out, dir)

			if err != nil {
				return err
			}
		}
	}
	log.Success("selected pack: " + lpack + "\n")
	chartsDir := filepath.Join(dir, "charts")
	jenkinsfileExists, err := util.FileExists(jenkinsfile)
	exists, err := util.FileExists(chartsDir)
	if exists && err == nil {
		exists, err = util.FileExists(filepath.Join(dir, "Dockerfile"))
		if exists && err == nil {
			if jenkinsfileExists || o.DisableJenkinsfileCheck {
				log.Warn("existing Dockerfile, Jenkinsfile and charts folder found so skipping 'draft create' step\n")
				return nil
			}
		}
	}
	jenknisfileBackup := ""
	if jenkinsfileExists && o.InitialisedGit && !o.DisableJenkinsfileCheck {
		// lets copy the old Jenkinsfile in case we overwrite it
		jenknisfileBackup = jenkinsfile + jenkinsfileBackupSuffix
		err = util.RenameFile(jenkinsfile, jenknisfileBackup)
		if err != nil {
			return fmt.Errorf("Failed to rename old Jenkinsfile: %s", err)
		}
	}

	err = pack.CreateFrom(dir, lpack)
	if err != nil {
		// lets ignore draft errors as sometimes it can't find a pack - e.g. for environments
		o.warnf("Failed to run draft create in %s due to %s", dir, err)
	}

	if jenknisfileBackup != "" {
		// if there's no Jenkinsfile created then rename it back again!
		jenkinsfileExists, err = util.FileExists(jenkinsfile)
		if err != nil {
			o.warnf("Failed to check for Jenkinsfile %s", err)
		} else {
			if jenkinsfileExists {
				if !o.InitialisedGit {
					err = os.Remove(jenknisfileBackup)
					if err != nil {
						o.warnf("Failed to remove Jenkinsfile backup %s", err)
					}
				}
			} else {
				// lets put the old one back again
				err = util.RenameFile(jenknisfileBackup, jenkinsfile)
				if err != nil {
					return fmt.Errorf("Failed to rename Jenkinsfile backup file: %s", err)
				}
			}
		}
	}

	// lets rename the chart to be the same as our app name
	err = o.renameChartToMatchAppName()
	if err != nil {
		return err
	}

	if o.PostDraftPackCallback != nil {
		err = o.PostDraftPackCallback()
		if err != nil {
			return err
		}
	}

	gitServerName, err := gits.GetHost(o.GitProvider)
	if err != nil {
		return err
	}

	currentUser := o.getCurrentUser()
	org := o.getOrganisation()
	if org == "" {
		org = currentUser
	}

	err = o.replacePlaceholders(gitServerName, org)
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

func (o *ImportOptions) getCurrentUser() string {
	//walk through every file in the given dir and update the placeholders
	currentUser := o.GitServer.CurrentUser
	if currentUser == "" {
		if o.GitProvider != nil {
			currentUser = o.GitProvider.CurrentUsername()
		}
	}
	if currentUser == "" {
		currentUser = o.Organisation
	}
	if currentUser == "" {
		o.warnf("No username defined for the current git server!")
		currentUser = o.DefaultOwner
	}
	return currentUser
}

func (o *ImportOptions) getOrganisation() string {
	org := ""
	gitInfo, err := gits.ParseGitURL(o.RepoURL)
	if err == nil && gitInfo.Organisation != "" {
		org = gitInfo.Organisation
	} else {
		org = o.Organisation
	}
	return org
}

func (o *ImportOptions) CreateNewRemoteRepository() error {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	dir := o.Dir
	_, defaultRepoName := filepath.Split(dir)

	if o.Organisation != "" {
		o.GitRepositoryOptions.Owner = o.Organisation
	}
	details, err := gits.PickNewGitRepository(o.Out, o.BatchMode, authConfigSvc, defaultRepoName, &o.GitRepositoryOptions, o.GitServer, o.GitUserAuth)
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
	o.InitialisedGit = true
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
		jclient, err := o.JenkinsClient()
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

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	jenkinsfile := o.Jenkinsfile
	if jenkinsfile == "" {
		jenkinsfile = jenkins.DefaultJenkinsfile
	}
	return o.ImportProject(gitURL, o.Dir, jenkinsfile, o.BranchPattern, o.Credentials, false, gitProvider, authConfigSvc, false, o.BatchMode)
}

func (o *ImportOptions) replacePlaceholders(gitServerName, gitOrg string) error {
	gitOrg = kube.ToValidName(strings.ToLower(gitOrg))
	o.Printf("replacing placeholders in directory %s\n", o.Dir)
	o.Printf("app name: %s, git server: %s, org: %s\n", o.AppName, gitServerName, gitOrg)

	if err := filepath.Walk(o.Dir, func(f string, fi os.FileInfo, err error) error {
		if fi.IsDir() && (fi.Name() == ".git" || fi.Name() == "node_modules" || fi.Name() == "vendor" || fi.Name() == "target") {
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
				line = strings.Replace(line, PlaceHolderGitProvider, gitServerName, -1)
				line = strings.Replace(line, PlaceHolderOrg, gitOrg, -1)
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
			return fmt.Errorf("error getting %s secret %v", name, err)
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
	chartsDir := filepath.Join(dir, "charts")
	files, err := ioutil.ReadDir(chartsDir)
	if err != nil {
		return fmt.Errorf("error matching a jenkins x draft pack name with chart folder %v", err)
	}
	for _, fi := range files {
		if fi.IsDir() {
			name := fi.Name()
			// TODO we maybe need to try check if the sub dir named after the build pack matches first?
			if name != "preview" && name != ".git" {
				oldChartsDir = filepath.Join(chartsDir, name)
				break
			}
		}
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

func (o *ImportOptions) fixDockerIgnoreFile() error {
	filename := filepath.Join(o.Dir, ".dockerignore")
	exists, err := util.FileExists(filename)
	if err == nil && exists {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("Failed to load %s: %s", filename, err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "Dockerfile" {
				lines = append(lines[:i], lines[i+1:]...)
				text := strings.Join(lines, "\n")
				err = ioutil.WriteFile(filename, []byte(text), DefaultWritePermissions)
				if err != nil {
					return err
				}
				o.Printf("Removed old `Dockerfile` entry from %s\n", util.ColorInfo(filename))
			}
		}
	}
	return nil
}

func (o *ImportOptions) fixMaven() error {
	if o.DisableMaven {
		return nil
	}
	dir := o.Dir
	pomName := filepath.Join(dir, "pom.xml")
	exists, err := util.FileExists(pomName)
	if err != nil {
		return err
	}
	if exists {
		// lets ensure the mvn plugins are ok
		out, err := o.getCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:plugin", "-Dartifact=maven-deploy-plugin", "-Dversion="+minimumMavenDeployVersion)
		if err != nil {
			return fmt.Errorf("Failed to update maven plugin: %s output: %s", err, out)
		}
		if !o.DryRun {
			err = gits.GitAdd(dir, "pom.xml")
			if err != nil {
				return err
			}
			err = gits.GitCommitIfChanges(dir, "fix:(plugins) use a better version of maven deploy plugin")
			if err != nil {
				return err
			}
		}

		// lets ensure the probe paths are ok
		out, err = o.getCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:chart")
		if err != nil {
			return fmt.Errorf("Failed to update chart: %s output: %s", err, out)
		}
		if !o.DryRun {
			err = gits.GitAdd(dir, "charts")
			if err != nil {
				return err
			}
			err = gits.GitCommitIfChanges(dir, "fix:(chart) fix up the probe path")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func allDraftPacks() ([]string, error) {
	// lets make sure we have the latest draft packs
	initOpts := InitOptions{
		CommonOptions: CommonOptions{},
	}
	log.Info("Getting latest packs ...\n")
	dir, err := initOpts.initBuildPacks()
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, f := range files {
		if f.IsDir() {
			result = append(result, f.Name())
		}
	}
	return result, err

}
