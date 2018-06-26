package cmd

import (
	"io"
	"strings"

	"fmt"

	os_user "os/user"

	"strconv"

	"errors"

	"path/filepath"

	"os"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"io/ioutil"
	"time"
)

type Cluster struct {
	Name     string
	Provider string
}

type GKECluster struct {
	Cluster
	ProjectId     string
	Zone          string
	MachineType   string
	MinNumOfNodes string
	MaxNumOfNodes string
	DiskSize      string
	AutoRepair    bool
	AutoUpgrade   bool
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
	Terraform             = "terraform"
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
		err = o.Git().AddRemote(dir, o.Flags.ForkOrganisationGitRepo, "upstream")
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
		path := filepath.Join(dir, Clusters, c.Name, Terraform)
		exists, err := util.FileExists(path)
		if err != nil {
			return fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
		}
		if !exists {
			os.MkdirAll(path, DefaultWritePermissions)

			switch c.Provider {
			case "gke":
				o.Git().Clone(TerraformTemplatesGKE, path)
				o.configureGKECluster(c, path)
			case "aks":
				// TODO add aks terraform templates URL
				return fmt.Errorf("creating an AKS cluster via terraform is not currently supported")
			case "eks":
				// TODO add eks terraform templates URL
				return fmt.Errorf("creating an EKS cluster via terraform is not currently supported")
			default:
				return fmt.Errorf("unknown kubernetes provider type %s must be one of %v", c.Provider, validTerraformClusterProviders)
			}
			os.RemoveAll(filepath.Join(path, ".git"))
			os.RemoveAll(filepath.Join(path, ".gitignore"))
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

func (o *CreateTerraformOptions) configureGKECluster(c Cluster, path string) error {
	gkeCluster := GKECluster{
		Cluster: c,
		DiskSize: "100",
		AutoUpgrade: false,
		AutoRepair: false,
	}

	if gkeCluster.ProjectId == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		gkeCluster.ProjectId = projectId
	}

	if gkeCluster.Name == "" {
		gkeCluster.Name = strings.ToLower(randomdata.SillyName())
		log.Infof("No cluster name provided so using a generated one: %s\n", gkeCluster.Name)
	}

	zone := gkeCluster.Zone
	if zone == "" {
		availableZones, err := gke.GetGoogleZones()
		if err != nil {
			return err
		}
		prompts := &survey.Select{
			Message:  "Google Cloud Zone:",
			Options:  availableZones,
			PageSize: 10,
			Help:     "The compute zone (e.g. us-central1-a) for the cluster",
		}

		err = survey.AskOne(prompts, &zone, nil)
		if err != nil {
			return err
		}
	}

	machineType := gkeCluster.MachineType
	if machineType == "" {
		prompts := &survey.Select{
			Message:  "Google Cloud Machine Type:",
			Options:  gke.GetGoogleMachineTypes(),
			Help:     "We recommend a minimum of n1-standard-2 for Jenkins X,  a table of machine descriptions can be found here https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture",
			PageSize: 10,
			Default:  "n1-standard-2",
		}

		err := survey.AskOne(prompts, &machineType, nil)
		if err != nil {
			return err
		}
	}

	minNumOfNodes := gkeCluster.MinNumOfNodes
	if minNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Minimum number of Nodes",
			Default: "3",
			Help:    "We recommend a minimum of 3 for Jenkins X,  the minimum number of nodes to be created in each of the cluster's zones",
		}

		survey.AskOne(prompt, &minNumOfNodes, nil)
	}

	maxNumOfNodes := gkeCluster.MaxNumOfNodes
	if maxNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Maximum number of Nodes",
			Default: "5",
			Help:    "We recommend at least 5 for Jenkins X,  the maximum number of nodes to be created in each of the cluster's zones",
		}

		survey.AskOne(prompt, &maxNumOfNodes, nil)
	}

	user, err := os_user.Current()
	if err != nil {
		return err
	}
	username := sanitizeLabel(user.Username)

	terraformVars := filepath.Join(path, "terraform.tfvars")
	o.writeKeyValueIfNotExists(terraformVars, "created_by", username)
	o.writeKeyValueIfNotExists(terraformVars, "created_timestamp", time.Now().Format("20060102150405"))
	//o.writeKeyValueIfNotExists(terraformVars, "credentials", keyPath)
	o.writeKeyValueIfNotExists(terraformVars, "cluster_name", gkeCluster.Name)
	o.writeKeyValueIfNotExists(terraformVars, "gcp_zone", zone)
	o.writeKeyValueIfNotExists(terraformVars, "gcp_project", gkeCluster.ProjectId)
	o.writeKeyValueIfNotExists(terraformVars, "min_node_count", gkeCluster.MinNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "max_node_count", gkeCluster.MaxNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "node_machine_type", gkeCluster.MachineType)
	o.writeKeyValueIfNotExists(terraformVars, "node_preemptible", "false")
	o.writeKeyValueIfNotExists(terraformVars, "node_disk_size", gkeCluster.DiskSize)
	o.writeKeyValueIfNotExists(terraformVars, "auto_repair", strconv.FormatBool(gkeCluster.AutoRepair))
	o.writeKeyValueIfNotExists(terraformVars, "auto_upgrade", strconv.FormatBool(gkeCluster.AutoUpgrade))
	o.writeKeyValueIfNotExists(terraformVars, "enable_kubernetes_alpha", "false")
	o.writeKeyValueIfNotExists(terraformVars, "enable_legacy_abac", "true")
	o.writeKeyValueIfNotExists(terraformVars, "logging_service", "logging.googleapis.com")
	o.writeKeyValueIfNotExists(terraformVars, "monitoring_service", "monitoring.googleapis.com")

	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (o *CreateTerraformOptions) getGoogleProjectId() (string, error) {
	existingProjects, err := gke.GetGoogleProjects()
	if err != nil {
		return "", err
	}

	var projectId string
	if len(existingProjects) == 0 {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("No existing Google Projects exist, create one now?"),
			Default: true,
		}
		flag := true
		err = survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return "", err
		}
		if !flag {
			return "", errors.New("no google project to create cluster in, please manual create one and rerun this wizard")
		}

		if flag {
			return "", errors.New("auto creating projects not yet implemented, please manually create one and rerun the wizard")
		}
	} else if len(existingProjects) == 1 {
		projectId = existingProjects[0]
		log.Infof("Using the only Google Cloud Project %s to create the cluster\n", util.ColorInfo(projectId))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		err := survey.AskOne(prompts, &projectId, nil)
		if err != nil {
			return "", err
		}
	}

	if projectId == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}

func (o *CreateTerraformOptions) writeLineIfNotExists(path string, line string) error {
	// file exists
	if _, err := os.Stat(path); err == nil {
		buffer, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(buffer)

		o.Debugf("Checking if %s contains '%s'\n", path, line)

		if strings.Contains(contents, line) {
			o.Debugf("Skipping %s\n", line)
			return nil
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	o.Debugf("Writing '%s' to %s\n", line, path)

	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}

func (o *CreateTerraformOptions) writeKeyValueIfNotExists(path string, key string, value string) error {
	// file exists
	if _, err := os.Stat(path); err == nil {
		buffer, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(buffer)

		o.Debugf("Checking if %s contains %s\n", path, key)

		if strings.Contains(contents, key) {
			o.Debugf("Skipping %s\n", key)
			return nil
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	line := fmt.Sprintf("%s = \"%s\"\n", key, value)
	o.Debugf("Writing '%s' to %s\n", line, path)

	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}
