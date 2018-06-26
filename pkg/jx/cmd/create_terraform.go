package cmd

import (
	"io"
	"strings"

	"fmt"

	"strconv"

	"errors"

	"path/filepath"

	"os"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

type Cluster struct {
	Name     string
	Provider string
}

type Flags struct {
	Cluster                 []string
	OrganisationRepoName    string
	ForkOrganisationGitRepo string
}

// CreateTerraformOptions the options for the create spring command
type CreateTerraformOptions struct {
	CreateOptions
	Flags                Flags
	Clusters             []Cluster
	GitRepositoryOptions gits.GitRepositoryOptions
}

var (
	validTerraformClusterProviders = []string{"gke"}

	createTerraformExample = templates.Examples(`
		jx create terraform

		# to specify the clusters via flags
		jx create terraform -c dev=gke -c stage=gke -c prod=gke

`)
)

const (
	Clusters              = "clusters"
	TerraformTemplatesGKE = "https://github.com/jenkins-x/terraform-jx-templates-gke.git"
)

// NewCmdCreateTerraform creates a command object for the "create" command
func NewCmdCreateTerraform(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateTerraformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Creates a Jenkins X terraform plan",
		Example: createTerraformExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd)
	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)

	return cmd
}

func (options *CreateTerraformOptions) addFlags(cmd *cobra.Command) {

	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, "cluster", "c", []string{}, "Name and Kubernetes provider (gke, aks, eks) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().StringVarP(&options.Flags.OrganisationRepoName, "organisation-repo-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details")
	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation git repos")

}

func stringInValidProviders(a string) bool {
	for _, b := range validTerraformClusterProviders {
		if b == a {
			return true
		}
	}
	return false
}

// Run implements this command
func (o *CreateTerraformOptions) Run() error {

	if len(o.Flags.Cluster) > 1 {
		err := o.validateClusterDetails()
		if err != nil {
			return err
		}
	}

	if len(o.Flags.Cluster) == 0 {
		err := o.ClusterDetailsWizard()
		if err != nil {
			return err
		}
	}

	log.Infof("Creating clusters %v", o.Clusters)

	err := o.createOrganisationGitRepo()
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateTerraformOptions) ClusterDetailsWizard() error {
	var numOfClusters int
	numOfClustersStr := "1"
	prompts := &survey.Input{
		Message: fmt.Sprintf("How many clusters shall we create?"),
		Default: numOfClustersStr,
	}

	err := survey.AskOne(prompts, &numOfClustersStr, nil)
	if err != nil {
		return err
	}
	numOfClusters, err = strconv.Atoi(numOfClustersStr)
	if err != nil {
		return err
	}

	for i := 1; i <= numOfClusters; i++ {
		c := Cluster{}

		defaultOption := ""
		if i == 1 {
			defaultOption = "dev"
		}
		prompts := &survey.Input{
			Message: fmt.Sprintf("Cluster %v name:", i),
			Default: defaultOption,
		}
		validator := survey.Required
		err := survey.AskOne(prompts, &c.Name, validator)
		if err != nil {
			return err
		}

		prompts = &survey.Input{
			Message: fmt.Sprintf("Cluster %v provider:", i),
			Default: "gke",
		}
		validator = survey.Validator(
			func(val interface{}) error {
				if str, ok := val.(string); !ok || !stringInValidProviders(str) {
					return errors.New(fmt.Sprintf("invalid cluster provider type %s, must be one of %v", str, validTerraformClusterProviders))
				}
				return nil
			},
		)
		err = survey.AskOne(prompts, &c.Provider, validator)
		if err != nil {
			return err
		}

		o.Clusters = append(o.Clusters, c)
	}
	return nil
}

func (o *CreateTerraformOptions) validateClusterDetails() error {
	for _, p := range o.Flags.Cluster {
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			return errors.New("need to provide cluster values as --cluster name=provider, e.g. --cluster production=gke")
		}
		if !stringInValidProviders(pair[1]) {
			return errors.New(fmt.Sprintf("invalid cluster provider type %s, must be one of %v", p, validTerraformClusterProviders))
		}
		c := Cluster{
			Name:     pair[0],
			Provider: pair[1],
		}
		o.Clusters = append(o.Clusters, c)
	}
	return nil
}

func (o *CreateTerraformOptions) createOrganisationGitRepo() error {

	organisationDir, err := util.OrganisationsDir()
	if err != nil {
		return err
	}

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	if o.Flags.OrganisationRepoName == "" {
		o.Flags.OrganisationRepoName = strings.ToLower(randomdata.SillyName())
	}
	defaultRepoName := fmt.Sprintf("organisation-%s", o.Flags.OrganisationRepoName)
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
	envDir := filepath.Join(organisationDir, owner)
	provider := details.GitProvider
	repo, err := provider.GetRepository(owner, repoName)
	if err == nil {
		fmt.Fprintf(o.Stdout(), "git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))
		// if the repo already exists then lets just modify it if required
		dir, err := util.CreateUniqueDirectory(envDir, details.RepoName, util.MaximumNewDirectoryAttempts)
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

		// add any new clusters
		err = o.createOrganisationFolderStructure(dir)
		if err != nil {
			return err
		}

		err = o.commitClusters(dir)
		if err != nil {
			return err
		}

		err = o.Git().PushMaster(dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Stdout(), "Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	} else {
		fmt.Fprintf(o.Stdout(), "Creating git repository %s/%s\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		repo, err = details.CreateRepository()
		if err != nil {
			return err
		}

		dir, err := util.CreateUniqueDirectory(organisationDir, details.RepoName, util.MaximumNewDirectoryAttempts)
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

		// create directory structure
		err = o.createOrganisationFolderStructure(dir)
		if err != nil {
			return err
		}

		err = o.commitClusters(dir)
		if err != nil {
			return err
		}

		err = o.Git().PushMaster(dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Stdout(), "Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

	}
	return nil
}
func (o *CreateTerraformOptions) createOrganisationFolderStructure(dir string) error {

	for _, c := range o.Clusters {
		path := filepath.Join(dir, Clusters, c.Name)
		exists, err := util.FileExists(path)
		if err != nil {
			return fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
		}
		if !exists {
			os.MkdirAll(path, DefaultWritePermissions)

			switch c.Provider {
			case "gke":
				o.Git().Clone(TerraformTemplatesGKE, path)
			case "aks":
				// TODO add aks terraform templates URL
			case "eks":
				// TODO add eks terraform templates URL
			default:
				return fmt.Errorf("unknown kubernetes provider type %s must be one of %v", c.Provider, validTerraformClusterProviders)
			}
			os.RemoveAll(filepath.Join(path, ".git"))
		}

	}

	return nil
}

func (o *CreateTerraformOptions) commitClusters(dir string) error {
	err := o.Git().Add(dir, "*")
	if err != nil {
		return err
	}
	changes, err := o.Git().HasChanges(dir)
	if err != nil {
		return err
	}
	if changes {
		return o.Git().CommitDir(dir, "Add organisation clusters")
	}
	return nil
}
