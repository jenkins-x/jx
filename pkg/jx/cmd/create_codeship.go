package cmd

import (
	"io"

	"fmt"

	"context"
	"errors"
	"github.com/codeship/codeship-go"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"io/ioutil"
	"strings"
	"github.com/jenkins-x/jx/pkg/version"
)

type CreateCodeshipFlags struct {
	Cluster              []string
	OrganisationName     string
	CodeshipUsername     string
	CodeshipPassword     string
	CodeshipOrganisation string
	GitUser              string
	GitEmail             string
	GKEServiceAccount    string
}

// CreateCodeshipOptions the options for the create spring command
type CreateCodeshipOptions struct {
	CreateOptions
	Flags                CreateCodeshipFlags
	GitRepositoryOptions gits.GitRepositoryOptions
}

var (
	createCodeshipExample = templates.Examples(`
		jx create codeship

		# to specify the clusters via flags
		jx create codeship -o org

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
	}

	cmd := &cobra.Command{
		Use:     "codeship",
		Short:   "Creates a Codeship build to apply ",
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

	return cmd
}

func (options *CreateCodeshipOptions) addFlags(cmd *cobra.Command) {
	// global flags
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, "cluster", "c", []string{}, "Name and Kubernetes provider (gke, aks, eks) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")

	cmd.Flags().StringVarP(&options.Flags.CodeshipUsername, "codeship-username", "", "", "The username to login to Codeship with, this will not be stored anywhere")
	cmd.Flags().StringVarP(&options.Flags.CodeshipPassword, "codeship-password", "", "", "The password to login to Codeship with, this will not be stored anywhere")
	cmd.Flags().StringVarP(&options.Flags.CodeshipOrganisation, "codeship-organisation", "", "", "The Codeship organisation to use, this will not be stored anywhere")

	cmd.Flags().StringVarP(&options.Flags.GitUser, "git-user", "", "Codeship", "The name to use for any git commits")
	cmd.Flags().StringVarP(&options.Flags.GitEmail, "git-email", "", "codeship@jenkins-x.io", "The email to use for any git commits")

	cmd.Flags().StringVarP(&options.Flags.GKEServiceAccount, "gke-service-account", "", "", "The GKE service account to use")
}

func (o *CreateCodeshipOptions) validate() error {

	if len(o.Flags.Cluster) == 0 {
		return errors.New("No cluster details provided")
	}

	err := o.validateClusterDetails()
	if err != nil {
		return err
	}

	if o.Flags.OrganisationName == "" {
		return errors.New("No organisation has been set")
	}

	if o.Flags.GKEServiceAccount == "" {
		return errors.New("No gke service account has been set")
	}
	return nil
}

func (o *CreateCodeshipOptions) validateClusterDetails() error {
	for _, p := range o.Flags.Cluster {
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			return errors.New("need to provide cluster values as --cluster name=provider, e.g. --cluster production=gke")
		}
		if !stringInValidProviders(pair[1]) {
			return errors.New(fmt.Sprintf("invalid cluster provider type %s, must be one of %v", p, validTerraformClusterProviders))
		}
	}
	return nil
}

// Run implements this command
func (o *CreateCodeshipOptions) Run() error {
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

	details, err := gits.PickNewGitRepository(o.Stdout(), o.BatchMode, authConfigSvc,
		defaultRepoName, &o.GitRepositoryOptions, nil, nil, o.Git())
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
	var dir string
	if err == nil {
		fmt.Fprintf(o.Stdout(), "git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))
		// if the repo already exists then lets just modify it if required
		dir, err = util.CreateUniqueDirectory(organisationDir, details.RepoName, util.MaximumNewDirectoryAttempts)
		if err != nil {
			return err
		}
		pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
		if err != nil {
			return err
		}
		err = o.Git().Clone(pushGitURL, dir)
		if err != nil {
			return err
		}
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
				{Name: "ENVIRONMENTS", Value: strings.Join(o.Flags.Cluster, ",")},
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
				{Name: "ENVIRONMENTS", Value: strings.Join(o.Flags.Cluster, ",")},
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
		return "1.3.99"
	}
	return version.Version
}
