package cmd

import (
	"io"

	"fmt"

	"context"
	"errors"
	"github.com/Pallinder/go-randomdata"
	"github.com/codeship/codeship-go"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"io/ioutil"
	"path"
	"strings"
	"os"
	)

type CreateCodeshipFlags struct {
	OrganisationName        string
	ForkOrganisationGitRepo string
	CodeshipUsername        string
	CodeshipPassword        string
	CodeshipOrganisation    string
	GitUser                 string
	GitEmail                string
	GKEServiceAccount       string
}

// CreateCodeshipOptions the options for the create spring command
type CreateCodeshipOptions struct {
	CreateOptions
	CreateTerraformOptions
	CreateGkeServiceAccountOptions
	Flags                CreateCodeshipFlags
	GitRepositoryOptions gits.GitRepositoryOptions
}

var (
	createCodeshipExample = templates.Examples(`
		jx create codeship

		# to specify the org and service account via flags
		jx create codeship -o org --gke-service-account <path>

`)
)

// NewCmdCreateCodeship creates a command object for the "create" command
func NewCmdCreateCodeship(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateCodeshipOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
		CreateTerraformOptions: CreateTerraformOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
			InstallOptions: createInstallOptions(f, out, errOut),
		},
		CreateGkeServiceAccountOptions: CreateGkeServiceAccountOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "codeship",
		Short:   "Creates a build on CodeShip to create/update JX clusters",
		Example: createCodeshipExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd)

	options.CreateGkeServiceAccountOptions.addFlags(cmd)
	options.CreateTerraformOptions.addFlags(cmd)

	return cmd
}

func (options *CreateCodeshipOptions) addFlags(cmd *cobra.Command) {
	// global flags
	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")

	cmd.Flags().StringVarP(&options.Flags.CodeshipUsername, "codeship-username", "", "", "The username to login to Codeship with, this will not be stored anywhere")
	cmd.Flags().StringVarP(&options.Flags.CodeshipPassword, "codeship-password", "", "", "The password to login to Codeship with, this will not be stored anywhere")
	cmd.Flags().StringVarP(&options.Flags.CodeshipOrganisation, "codeship-organisation", "", "", "The Codeship organisation to use, this will not be stored anywhere")

	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation git repos")

	cmd.Flags().StringVarP(&options.Flags.GitUser, "git-user", "", "Codeship", "The name to use for any git commits")
	cmd.Flags().StringVarP(&options.Flags.GitEmail, "git-email", "", "codeship@jenkins-x.io", "The email to use for any git commits")

	cmd.Flags().StringVarP(&options.Flags.GKEServiceAccount, "gke-service-account", "", "", "The GKE service account to use")
}

func (o *CreateCodeshipOptions) validate() error {
	if o.Flags.OrganisationName == "" {
		return errors.New("No organisation has been set")
	}

	// TODO we should only do this if a GKE cluster has been specified
	if o.Flags.GKEServiceAccount == "" {
		return errors.New("No gke service account has been set")
	}
	return nil
}

// Run implements this command
func (o *CreateCodeshipOptions) Run() error {
	if o.Flags.OrganisationName == "" {
		o.Flags.OrganisationName = strings.ToLower(randomdata.SillyName())
	}

	// if gke-service-account is not set, create the service account
	if o.Flags.GKEServiceAccount == "" {
		gkeServiceAccountPath := path.Join(util.HomeDir(), fmt.Sprintf("%s.key.json", o.Flags.OrganisationName))

		o.CreateGkeServiceAccountOptions.Flags.Name = o.Flags.OrganisationName
		o.CreateGkeServiceAccountOptions.CommonOptions.BatchMode = o.CreateOptions.CommonOptions.BatchMode
		err := o.CreateGkeServiceAccountOptions.Run()
		if err != nil {
			return err
		}

		o.Flags.GKEServiceAccount = gkeServiceAccountPath
	}

	err := o.validate()
	if err != nil {
		return err
	}

	if o.Flags.CodeshipUsername == "" {
		prompt := &survey.Input{
			Message: "Username for Codeship",
			Help:    "This will not be stored anywhere",
		}

		err := survey.AskOne(prompt, &o.Flags.CodeshipUsername, nil)
		if err != nil {
			return err
		}
	}

	if o.Flags.CodeshipPassword == "" {
		prompt := &survey.Password{
			Message: "Password for Codeship",
			Help:    "This will not be stored anywhere",
		}

		err := survey.AskOne(prompt, &o.Flags.CodeshipPassword, nil)
		if err != nil {
			return err
		}
	}

	if o.Flags.CodeshipOrganisation == "" {
		prompt := &survey.Input{
			Message: "Codeship organisation",
			Help:    "This will not be stored anywhere",
		}

		err := survey.AskOne(prompt, &o.Flags.CodeshipOrganisation, nil)
		if err != nil {
			return err
		}
	}

	organisationDir, err := util.OrganisationsDir()
	if err != nil {
		return err
	}

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	defaultRepoName := fmt.Sprintf("organisation-%s", o.Flags.OrganisationName)

	details, err := gits.PickNewOrExistingGitRepository(o.Stdout(), o.BatchMode, authConfigSvc,
		defaultRepoName, &o.GitRepositoryOptions, nil, nil, o.Git(), true)
	if err != nil {
		return err
	}
	org := details.Organisation
	repoName := details.RepoName
	owner := org
	if owner == "" {
		owner = details.User.Username
	}
	provider := details.GitProvider
	repo, err := provider.GetRepository(owner, repoName)
	remoteRepoExists := err == nil
	var dir string

	if !remoteRepoExists {
		fmt.Fprintf(o.Stdout(), "Creating git repository %s/%s\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		repo, err = details.CreateRepository()
		if err != nil {
			return err
		}

		dir, err = util.CreateUniqueDirectory(organisationDir, details.RepoName, util.MaximumNewDirectoryAttempts)
		if err != nil {
			return err
		}

		err = o.Git().Clone(o.Flags.ForkOrganisationGitRepo, dir)
		if err != nil {
			return err
		}
		pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
		if err != nil {
			return err
		}
		err = o.Git().AddRemote(dir, "upstream", o.Flags.ForkOrganisationGitRepo)
		if err != nil {
			return err
		}
		err = o.Git().SetRemoteURL(dir, "origin", pushGitURL)
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintf(o.Stdout(), "git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		dir = path.Join(organisationDir, details.RepoName)
		localDirExists, err := util.FileExists(dir)
		if err != nil {
			return err
		}

		if localDirExists {
			// if remote repo does exist & local does exist, git pull the local repo
			fmt.Fprintf(o.Stdout(), "local directory already exists\n")

			err = o.Git().Pull(dir)
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(o.Stdout(), "cloning repository locally\n")
			err = os.MkdirAll(dir, os.FileMode(0755))
			if err != nil {
				return err
			}

			// if remote repo does exist & local directory does not exist, clone locally
			pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
			if err != nil {
				return err
			}
			err = o.Git().Clone(pushGitURL, dir)
			if err != nil {
				return err
			}
		}
	}

	if !remoteRepoExists {
		o.CreateTerraformOptions.CommonOptions.BatchMode = o.CreateOptions.CommonOptions.BatchMode
		o.CreateTerraformOptions.Flags.OrganisationName = o.Flags.OrganisationName
		o.CreateTerraformOptions.Flags.SkipTerraformApply = true
		o.CreateTerraformOptions.Flags.GKEServiceAccount = o.Flags.GKEServiceAccount
		o.CreateTerraformOptions.Flags.LocalRepository = dir

		err = o.CreateTerraformOptions.Run()
		if err != nil {
			return err
		}
	}

	clusterDir := path.Join(dir, "clusters")
	clusters, err := findClusters(clusterDir)
	if err != nil {
		return err
	}

	auth := codeship.NewBasicAuth(o.Flags.CodeshipUsername, o.Flags.CodeshipPassword)
	client, err := codeship.New(auth)
	if err != nil {
		return err
	}

	ctx := context.Background()

	csOrg, err := client.Organization(ctx, o.Flags.CodeshipOrganisation)
	if err != nil {
		return err
	}

	_, uuid, err := ProjectExists(ctx, csOrg, o.Flags.CodeshipOrganisation, repoName)

	b, err := ioutil.ReadFile(o.Flags.GKEServiceAccount)
	if err != nil {
		return err
	}

	serviceAccount := string(b)

	if uuid == "" {
		createProjectRequest := codeship.ProjectCreateRequest{
			Type:          codeship.ProjectTypeBasic,
			RepositoryURL: fmt.Sprintf("git@github.com:%s/%s", owner, repoName),
			SetupCommands: []string{"./build.sh"},
			EnvironmentVariables: []codeship.EnvironmentVariable{
				{Name: "GKE_SA_JSON", Value: serviceAccount},
				{Name: "ORG", Value: o.Flags.OrganisationName},
				{Name: "GIT_USERNAME", Value: details.User.Username},
				{Name: "GIT_API_TOKEN", Value: details.User.ApiToken},
				{Name: "JX_VERSION", Value: jxVersion()},
				{Name: "GIT_USER", Value: o.Flags.GitUser},
				{Name: "GIT_EMAIL", Value: o.Flags.GitEmail},
				{Name: "BUILD_NUMBER", Value: "1"},
				{Name: "ENVIRONMENTS", Value: strings.Join(clusters, ",")},
			},
		}

		project, _, err := csOrg.CreateProject(ctx, createProjectRequest)

		if err != nil {
			return err
		}

		uuid = project.UUID

		log.Infof("Created Project %s\n", util.ColorInfo(project.Name))
	} else {
		updateProjectRequest := codeship.ProjectUpdateRequest{
			Type:          codeship.ProjectTypeBasic,
			SetupCommands: []string{"./build.sh"},
			EnvironmentVariables: []codeship.EnvironmentVariable{
				{Name: "GKE_SA_JSON", Value: serviceAccount},
				{Name: "ORG", Value: o.Flags.OrganisationName},
				{Name: "GIT_USERNAME", Value: details.User.Username},
				{Name: "GIT_API_TOKEN", Value: details.User.ApiToken},
				{Name: "JX_VERSION", Value: jxVersion()},
				{Name: "GIT_USER", Value: o.Flags.GitUser},
				{Name: "GIT_EMAIL", Value: o.Flags.GitEmail},
				{Name: "BUILD_NUMBER", Value: "1"},
				{Name: "ENVIRONMENTS", Value: strings.Join(clusters, ",")},
			},
		}

		project, _, err := csOrg.UpdateProject(ctx, uuid, updateProjectRequest)
		if err != nil {
			return err
		}
		log.Infof("Updated Project %s\n", util.ColorInfo(project.Name))
	}

	log.Infof("Triggering build for %s\n", util.ColorInfo(uuid))
	_, _, err = csOrg.CreateBuild(ctx, uuid, "heads/master", "")
	if err != nil {
		return err
	}

	return nil
}

func ProjectExists(ctx context.Context, org *codeship.Organization, codeshipOrg string, codeshipRepo string) (bool, string, error) {
	projects, _, err := org.ListProjects(ctx)
	if err != nil {
		return false, "", err
	}

	projectName := fmt.Sprintf("%s/%s", codeshipOrg, codeshipRepo)

	for _, p := range projects.Projects {
		if p.Name == projectName {
			log.Infof("Project %s already exists\n", util.ColorInfo(p.Name))
			return true, p.UUID, nil
		}
	}
	return false, "", nil
}

func jxVersion() string {
	if version.Version == "1.0.1" {
		return "1.3.132"
	}
	return version.Version
}

func findClusters(path string) ([]string, error) {
	var clusters = []string{}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return clusters, err
	}

	for _, f := range files {
		if f.IsDir() {
			clusters = append(clusters, fmt.Sprintf("%s=gke", f.Name()))
		}
	}
	return clusters, nil
}
