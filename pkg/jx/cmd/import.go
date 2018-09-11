package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/pkg/errors"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/jenkins-x/draft-repo/pkg/draft/pack"
	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/config"
	jxdraft "github.com/jenkins-x/jx/pkg/draft"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//_ "github.com/Azure/draft/pkg/linguist"
	"time"

	"github.com/denormal/go-gitignore"
	"github.com/jenkins-x/jx/pkg/prow"
)

const (
	PlaceHolderAppName           = "REPLACE_ME_APP_NAME"
	PlaceHolderGitProvider       = "REPLACE_ME_GIT_PROVIDER"
	PlaceHolderOrg               = "REPLACE_ME_ORG"
	PlaceHolderDockerRegistryOrg = "REPLACE_ME_DOCKER_REGISTRY_ORG"

	JenkinsfileBackupSuffix = ".backup"

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
	DockerRegistryOrg       string

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

func NewCmdImport(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
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
			CheckErr(err)
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
	cmd.Flags().StringVarP(&options.DockerRegistryOrg, "docker-registry-org", "", "", "The name of the docker registry organisation to use. If not specified then the git provider organisation will be used")

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
		log.Infoln("Available draft packs:")
		for i := 0; i < len(packs); i++ {
			log.Infof(packs[i] + "\n")
		}
		return nil
	}

	o.Factory.SetBatch(o.BatchMode)

	var err error
	isProw := false
	if !o.DryRun {
		_, _, err = o.KubeClient()
		if err != nil {
			return err
		}

		_, _, err = o.JXClient()
		if err != nil {
			return err
		}

		apisClient, err := o.CreateApiExtensionsClient()
		if err != nil {
			return err
		}
		err = kube.RegisterEnvironmentCRD(apisClient)
		if err != nil {
			return err
		}

		isProw, err = o.isProw()
		if err != nil {
			return err
		}

		if !isProw {
			o.Jenkins, err = o.JenkinsClient()
			if err != nil {
				return err
			}
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
		// Get the org in case there is more than one user auth on the server and batchMode is true
		org := o.getOrganisationOrCurrentUser()
		userAuth, err = config.PickServerUserAuth(server, "git user name:", o.BatchMode, org)
		if err != nil {
			return err
		}
		if server.Kind == "" {
			server.Kind, err = o.GitServerHostURLKind(server.URL)
			if err != nil {
				return err
			}
		}
		if userAuth.IsInvalid() {
			f := func(username string) error {
				o.Git().PrintCreateRepositoryGenerateAccessToken(server, username, o.Out)
				return nil
			}
			err = config.EditUserAuth(server.Label(), userAuth, userAuth.Username, true, o.BatchMode, f)
			if err != nil {
				return err
			}

			// TODO lets verify the auth works?
			if userAuth.IsInvalid() {
				return fmt.Errorf("Authentication has failed for user %v. Please check the user's access credentials and try again.\n", userAuth.Username)
			}
		}
		err = authConfigSvc.SaveUserAuth(server.URL, userAuth)
		if err != nil {
			return fmt.Errorf("Failed to store git auth configuration %s", err)
		}

		o.GitServer = server
		o.GitUserAuth = userAuth
		o.GitProvider, err = gits.CreateProvider(server, userAuth, o.Git())
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
	if o.AppName == "" {
		dir, err := filepath.Abs(o.Dir)
		if err != nil {
			return err
		}
		_, o.AppName = filepath.Split(dir)
	}
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
			err = o.Git().Push(o.Dir)
			if err != nil {
				return err
			}
		}
	}

	if o.DryRun {
		log.Infoln("dry-run so skipping import to Jenkins X")
		return nil
	}

	if !isProw {
		err = o.checkChartmuseumCredentialExists()
		if err != nil {
			return err
		}
	}

	return o.doImport()
}

func (o *ImportOptions) ImportProjectsFromGitHub() error {
	repos, err := gits.PickRepositories(o.GitProvider, o.Organisation, "Which repositories do you want to import", o.SelectAll, o.SelectFilter)
	if err != nil {
		return err
	}

	log.Infoln("Selected repositories")
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
		log.Infof("Importing repository %s\n", util.ColorInfo(r.Name))
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

	defaultJenkinsfile := filepath.Join(dir, jenkins.DefaultJenkinsfile)
	jenkinsfile := defaultJenkinsfile
	withRename := false
	if o.Jenkinsfile != "" {
		jenkinsfile = filepath.Join(dir, o.Jenkinsfile)
		withRename = true
	}
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
			pack, err := util.PomFlavour(pomName)
			if err != nil {
				return err
			}
			if len(pack) > 0 {
				if pack == util.LIBERTY {
					lpack = filepath.Join(packsDir, "liberty")
				} else if pack == util.APPSERVER {
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
	jenkinsfileBackup := ""
	if jenkinsfileExists && o.InitialisedGit && !o.DisableJenkinsfileCheck {
		// lets copy the old Jenkinsfile in case we overwrite it
		jenkinsfileBackup = jenkinsfile + JenkinsfileBackupSuffix
		err = util.RenameFile(jenkinsfile, jenkinsfileBackup)
		if err != nil {
			return fmt.Errorf("Failed to rename old Jenkinsfile: %s", err)
		}
	} else if withRename {
		defaultJenkinsfileExists, err := util.FileExists(defaultJenkinsfile)
		if defaultJenkinsfileExists && o.InitialisedGit && !o.DisableJenkinsfileCheck {
			jenkinsfileBackup = defaultJenkinsfile + JenkinsfileBackupSuffix
			err = util.RenameFile(defaultJenkinsfile, jenkinsfileBackup)
			if err != nil {
				return fmt.Errorf("Failed to rename old Jenkinsfile: %s", err)
			}

		}
	}

	err = pack.CreateFrom(dir, lpack)
	if err != nil {
		// lets ignore draft errors as sometimes it can't find a pack - e.g. for environments
		log.Warnf("Failed to run draft create in %s due to %s", dir, err)
	}

	unpackedDefaultJenkinsfile := defaultJenkinsfile
	if unpackedDefaultJenkinsfile != jenkinsfile {
		unpackedDefaultJenkinsfileExists := false
		unpackedDefaultJenkinsfileExists, err = util.FileExists(unpackedDefaultJenkinsfile)
		if unpackedDefaultJenkinsfileExists {
			err = util.RenameFile(unpackedDefaultJenkinsfile, jenkinsfile)
			if err != nil {
				return fmt.Errorf("Failed to rename Jenkinsfile file from '%s' to '%s': %s", unpackedDefaultJenkinsfile, jenkinsfile, err)
			}
			if jenkinsfileBackup != "" {
				err = util.RenameFile(jenkinsfileBackup, defaultJenkinsfile)
				if err != nil {
					return fmt.Errorf("Failed to rename Jenkinsfile backup file: %s", err)
				}
			}
		}
	} else if jenkinsfileBackup != "" {
		// if there's no Jenkinsfile created then rename it back again!
		jenkinsfileExists, err = util.FileExists(jenkinsfile)
		if err != nil {
			log.Warnf("Failed to check for Jenkinsfile %s", err)
		} else {
			if jenkinsfileExists {
				if !o.InitialisedGit {
					err = os.Remove(jenkinsfileBackup)
					if err != nil {
						log.Warnf("Failed to remove Jenkinsfile backup %s", err)
					}
				}
			} else {
				// lets put the old one back again
				err = util.RenameFile(jenkinsfileBackup, jenkinsfile)
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

	org := o.getOrganisationOrCurrentUser()
	dockerRegistryOrg := o.getDockerRegistryOrg()
	err = o.ReplacePlaceholders(gitServerName, org, dockerRegistryOrg)
	if err != nil {
		return err
	}
	err = o.Git().Add(dir, "*")
	if err != nil {
		return err
	}
	err = o.Git().CommitIfChanges(dir, "Draft create")
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) getDockerRegistryOrg() string {
	dockerRegistryOrg := o.DockerRegistryOrg
	if dockerRegistryOrg == "" {
		dockerRegistryOrg = o.getOrganisationOrCurrentUser()
	}
	return dockerRegistryOrg
}

func (o *ImportOptions) getOrganisationOrCurrentUser() string {
	currentUser := o.getCurrentUser()
	org := o.getOrganisation()
	if org == "" {
		org = currentUser
	}
	return org
}

func (o *ImportOptions) getCurrentUser() string {
	//walk through every file in the given dir and update the placeholders
	var currentUser string
	if o.Organisation != "" {
		return o.Organisation
	}
	if o.GitServer != nil {
		currentUser = o.GitServer.CurrentUser
		if currentUser == "" {
			if o.GitProvider != nil {
				currentUser = o.GitProvider.CurrentUsername()
			}
		}
	}
	if currentUser == "" {
		log.Warn("No username defined for the current git server!")
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
	details, err := gits.PickNewGitRepository(o.Out, o.BatchMode, authConfigSvc, defaultRepoName, &o.GitRepositoryOptions,
		o.GitServer, o.GitUserAuth, o.Git())
	if err != nil {
		return err
	}
	repo, err := details.CreateRepository()
	if err != nil {
		return err
	}
	o.GitProvider = details.GitProvider

	o.RepoURL = repo.CloneURL
	pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
	if err != nil {
		return err
	}
	err = o.Git().AddRemote(dir, "origin", pushGitURL)
	if err != nil {
		return err
	}
	err = o.Git().PushMaster(dir)
	if err != nil {
		return err
	}
	log.Infof("Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

	// If the user creating the repo is not the pipeline user, add the pipeline user as a contributor to the repo
	config := authConfigSvc.Config()
	if config.PipeLineUsername != o.GitUserAuth.Username && config.CurrentServer == config.PipeLineServer {
		// Make the invitation
		err := o.GitProvider.AddCollaborator(config.PipeLineUsername, details.RepoName)
		if err != nil {
			return err
		}

		// Create a new provider for the pipeline user
		pipelineUserAuth := config.FindUserAuth(config.CurrentServer, config.PipeLineUsername)
		pipelineServerAuth := config.GetServer(config.CurrentServer)
		pipelineUserProvider, err := gits.CreateProvider(pipelineServerAuth, pipelineUserAuth, o.Git())
		if err != nil {
			return err
		}

		// Get all invitations for the pipeline user
		// Wrapped in retry to not immediately fail the quickstart creation if APIs are flaky.
		f := func() error {
			invites, _, err := pipelineUserProvider.ListInvitations()
			if err != nil {
				return err
			}
			for _, x := range invites {
				// Accept all invitations for the pipeline user
				_, err = pipelineUserProvider.AcceptInvitation(*x.ID)
				if err != nil {
					return err
				}
			}
			return nil
		}
		exponentialBackOff := backoff.NewExponentialBackOff()
		timeout := 20 * time.Second
		exponentialBackOff.MaxElapsedTime = timeout
		exponentialBackOff.Reset()
		err = backoff.Retry(f, exponentialBackOff)
		if err != nil {
			return err
		}

	}

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
		return errors.Wrapf(err, "failed to create unique directory for '%s'", o.Dir)
	}
	err = o.Git().Clone(url, cloneDir)
	if err != nil {
		return errors.Wrapf(err, "failed to clone in directory '%s'", cloneDir)
	}
	o.Dir = cloneDir
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DiscoverGit() error {
	if !o.DisableDotGitSearch {
		root, gitConf, err := o.Git().FindGitConfigDir(o.Dir)
		if err != nil {
			return err
		}
		if root != "" {
			if root != o.Dir {
				log.Infof("Importing from directory %s as we found a .git folder there\n", root)
			}
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
		log.Infof("The directory %s is not yet using git\n", util.ColorInfo(dir))
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
	err := o.Git().Init(dir)
	if err != nil {
		return err
	}
	o.GitConfDir = filepath.Join(dir, ".git", "config")
	err = o.DefaultGitIgnore()
	if err != nil {
		return err
	}
	err = o.Git().Add(dir, ".gitignore")
	if err != nil {
		return err
	}
	err = o.Git().Add(dir, "*")
	if err != nil {
		return err
	}

	err = o.Git().Status(dir)
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
	err = o.Git().CommitIfChanges(dir, message)
	if err != nil {
		return err
	}
	log.Infof("\nGit repository created\n")
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
	url := o.Git().GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = o.Git().GetRemoteUrl(cfg, "upstream")
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

func (o *ImportOptions) doImport() error {
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

	err = o.ensureDockerRepositoryExists()
	if err != nil {
		return err
	}

	isProw, err := o.isProw()
	if err != nil {
		return err
	}
	if isProw {
		// register the webhook
		err = o.createWebhookProw(gitURL, gitProvider)
		if err != nil {
			return err
		}
		return o.addProwConfig(gitURL)
	}

	return o.ImportProject(gitURL, o.Dir, jenkinsfile, o.BranchPattern, o.Credentials, false, gitProvider, authConfigSvc, false, o.BatchMode)
}

func (o *ImportOptions) addProwConfig(gitURL string) error {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}
	repo := gitInfo.Organisation + "/" + gitInfo.Name
	return prow.AddApplication(o.KubeClientCached, []string{repo}, o.currentNamespace)
}

// ensureDockerRepositoryExists for some kinds of container registry we need to pre-initialise its use such as for ECR
func (o *ImportOptions) ensureDockerRepositoryExists() error {
	orgName := o.getOrganisationOrCurrentUser()
	appName := o.AppName
	if orgName == "" {
		log.Warnf("Missing organisation name!\n")
		return nil
	}
	if appName == "" {
		log.Warnf("Missing application name!\n")
		return nil
	}
	kubeClient, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, curNs)
	if err != nil {
		return err
	}

	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(kube.ConfigMapJenkinsDockerRegistry, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Could not find ConfigMap %s in namespace %s: %s", kube.ConfigMapJenkinsDockerRegistry, ns, err)
	}
	if cm.Data != nil {
		dockerRegistry := cm.Data["docker.registry"]
		if dockerRegistry != "" {
			if strings.HasSuffix(dockerRegistry, ".amazonaws.com") && strings.Index(dockerRegistry, ".ecr.") > 0 {
				return amazon.LazyCreateRegistry(orgName, appName)
			}
		}
	}
	return nil
}

// ReplacePlaceholders replaces git server name, git org, and docker registry org placeholders
func (o *ImportOptions) ReplacePlaceholders(gitServerName, gitOrg, dockerRegistryOrg string) error {
	gitOrg = kube.ToValidName(strings.ToLower(gitOrg))
	log.Infof("replacing placeholders in directory %s\n", o.Dir)
	log.Infof("app name: %s, git server: %s, org: %s, docker registry org: %s\n", o.AppName, gitServerName, gitOrg, dockerRegistryOrg)

	ignore, err := gitignore.NewRepository(o.Dir)
	if err != nil {
		return err
	}

	if err := filepath.Walk(o.Dir, func(f string, fi os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(o.Dir, f)
		match := ignore.Relative(relPath, fi.IsDir())
		matchIgnore := match != nil && match.Ignore() //Defaults to including if match == nil

		if fi.IsDir() {
			if matchIgnore || fi.Name() == ".git" {
				log.Infof("skipping directory %q\n", f)
				return filepath.SkipDir
			}
			return nil
		}

		// Dont process nor follow symlinks
		if (fi.Mode() & os.ModeSymlink) == os.ModeSymlink {
			log.Infof("skipping symlink file %q\n", f)
			return nil
		}

		if !matchIgnore {
			input, err := ioutil.ReadFile(f)
			if err != nil {
				log.Errorf("failed to read file %s: %v", f, err)
				return err
			}

			lines := strings.Split(string(input), "\n")

			for i, line := range lines {
				line = strings.Replace(line, PlaceHolderAppName, strings.ToLower(o.AppName), -1)
				line = strings.Replace(line, PlaceHolderGitProvider, strings.ToLower(gitServerName), -1)
				line = strings.Replace(line, PlaceHolderOrg, strings.ToLower(gitOrg), -1)
				line = strings.Replace(line, PlaceHolderDockerRegistryOrg, strings.ToLower(dockerRegistryOrg), -1)
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
		secret, err := o.KubeClientCached.CoreV1().Secrets(o.currentNamespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting %s secret %v", name, err)
		}

		data := secret.Data
		username := string(data["BASIC_AUTH_USER"])
		password := string(data["BASIC_AUTH_PASS"])

		err = o.retry(3, 10*time.Second, func() (err error) {
			return o.Jenkins.CreateCredential(name, username, password)
		})

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
				log.Infof("Removed old `Dockerfile` entry from %s\n", util.ColorInfo(filename))
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
		err = o.installMavenIfRequired()
		if err != nil {
			return err
		}

		// lets ensure the mvn plugins are ok
		out, err := o.getCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:plugin", "-Dartifact=maven-deploy-plugin", "-Dversion="+minimumMavenDeployVersion)
		if err != nil {
			return fmt.Errorf("Failed to update maven plugin: %s output: %s", err, out)
		}
		if !o.DryRun {
			err = o.Git().Add(dir, "pom.xml")
			if err != nil {
				return err
			}
			err = o.Git().CommitIfChanges(dir, "fix:(plugins) use a better version of maven deploy plugin")
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
			exists, err := util.FileExists(filepath.Join(dir, "charts"))
			if err != nil {
				return err
			}
			if exists {
				err = o.Git().Add(dir, "charts")
				if err != nil {
					return err
				}
				err = o.Git().CommitIfChanges(dir, "fix:(chart) fix up the probe path")
				if err != nil {
					return err
				}
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
