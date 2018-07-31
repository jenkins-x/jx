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
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/terraform"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"path"
	"time"
)

type Cluster interface {
	Name() string
	ClusterName() string
	Provider() string
	Context() string
	CreateTfVarsFile(path string) error
}

type GKECluster struct {
	_Name          string
	_Provider      string
	Organisation   string
	ProjectId      string
	Zone           string
	MachineType    string
	MinNumOfNodes  string
	MaxNumOfNodes  string
	DiskSize       string
	AutoRepair     bool
	AutoUpgrade    bool
	ServiceAccount string
}

func (g GKECluster) Name() string {
	return g._Name
}

func (g GKECluster) ClusterName() string {
	return fmt.Sprintf("%s-%s", g.Organisation, g._Name)
}

func (g GKECluster) Provider() string {
	return g._Provider
}

func (g GKECluster) Context() string {
	return fmt.Sprintf("%s_%s_%s_%s", g._Provider, g.ProjectId, g.Zone, g.ClusterName())
}

func (g GKECluster) Region() string {
	return gke.GetRegionFromZone(g.Zone)
}

func (g GKECluster) CreateTfVarsFile(path string) error {
	user, err := os_user.Current()
	var username string
	if err != nil {
		username = "unknown"
	} else {
		username = sanitizeLabel(user.Username)
	}

	err = terraform.WriteKeyValueToFileIfNotExists(path, "created_by", username)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "created_timestamp", time.Now().Format("20060102150405"))
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "cluster_name", g.ClusterName())
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "organisation", g.Organisation)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "provider", g._Provider)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "gcp_zone", g.Zone)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "gcp_project", g.ProjectId)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "min_node_count", g.MinNumOfNodes)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "max_node_count", g.MaxNumOfNodes)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "node_machine_type", g.MachineType)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "node_preemptible", "false")
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "node_disk_size", g.DiskSize)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "auto_repair", strconv.FormatBool(g.AutoRepair))
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "auto_upgrade", strconv.FormatBool(g.AutoUpgrade))
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "enable_kubernetes_alpha", "false")
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "enable_legacy_abac", "true")
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "logging_service", "logging.googleapis.com")
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "monitoring_service", "monitoring.googleapis.com")
	if err != nil {
		return err
	}
	return nil
}

func (g *GKECluster) ParseTfVarsFile(path string) {
	g.Zone, _ = terraform.ReadValueFromFile(path, "gcp_zone")
	g.Organisation, _ = terraform.ReadValueFromFile(path, "organisation")
	g._Provider, _ = terraform.ReadValueFromFile(path, "provider")
	g.ProjectId, _ = terraform.ReadValueFromFile(path, "gcp_project")
	g.MinNumOfNodes, _ = terraform.ReadValueFromFile(path, "min_node_count")
	g.MaxNumOfNodes, _ = terraform.ReadValueFromFile(path, "max_node_count")
	g.MachineType, _ = terraform.ReadValueFromFile(path, "node_machine_type")
	g.DiskSize, _ = terraform.ReadValueFromFile(path, "node_disk_size")

	autoRepair, _ := terraform.ReadValueFromFile(path, "auto_repair")
	b, _ := strconv.ParseBool(autoRepair)
	g.AutoRepair = b

	autoUpgrade, _ := terraform.ReadValueFromFile(path, "auto_upgrade")
	b, _ = strconv.ParseBool(autoUpgrade)
	g.AutoUpgrade = b
}

type Flags struct {
	Cluster                     []string
	OrganisationName            string
	ForkOrganisationGitRepo     string
	SkipTerraformApply          bool
	IgnoreTerraformWarnings     bool
	JxEnvironment               string
	GKEProjectId                string
	GKESkipEnableApis           bool
	GKEZone                     string
	GKEMachineType              string
	GKEMinNumOfNodes            string
	GKEMaxNumOfNodes            string
	GKEDiskSize                 string
	GKEAutoRepair               bool
	GKEAutoUpgrade              bool
	GKEServiceAccount           string
	LocalOrganisationRepository string
}

// CreateTerraformOptions the options for the create spring command
type CreateTerraformOptions struct {
	CreateOptions
	InstallOptions       InstallOptions
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
	validTerraformVersions = "0.11.0"

	gkeBucketConfiguration = `terraform {
  required_version = ">= %s"
  backend "gcs" {
    bucket      = "%s-%s-terraform-state"
    prefix      = "%s"
  }
}`
)

const (
	Clusters              = "clusters"
	Terraform             = "terraform"
	TerraformTemplatesGKE = "https://github.com/jenkins-x/terraform-jx-templates-gke.git"
)

// NewCmdCreateTerraform creates a command object for the "create" command
func NewCmdCreateTerraform(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateTerraformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
		InstallOptions: createInstallOptions(f, out, errOut),
	}

	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Creates a Jenkins X terraform plan",
		Example: createTerraformExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")
	cmd.Flags().StringVarP(&options.Flags.GKEServiceAccount, "gke-service-account", "", "", "The service account to use to connect to GKE")
	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation git repos")

	return cmd
}

func (options *CreateTerraformOptions) addFlags(cmd *cobra.Command) {
	// global flags
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, "cluster", "c", []string{}, "Name and Kubernetes provider (gke, aks, eks) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().BoolVarP(&options.Flags.SkipTerraformApply, "skip-terraform-apply", "", false, "Skip applying the generated terraform plans")
	cmd.Flags().BoolVarP(&options.Flags.IgnoreTerraformWarnings, "ignore-terraform-warnings", "", false, "Ignore any warnings about the terraform plan being potentially destructive")
	cmd.Flags().StringVarP(&options.Flags.JxEnvironment, "jx-environment", "", "dev", "The cluster name to install jx inside")
	cmd.Flags().StringVarP(&options.Flags.LocalOrganisationRepository, "local-organisation-repository", "", "", "Rather than cloning from a remote git server, the local directory to use for the organisational folder")

	// gke specific overrides
	cmd.Flags().StringVarP(&options.Flags.GKEDiskSize, "gke-disk-size", "", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoUpgrade, "gke-enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoRepair, "gke-enable-autorepair", "", true, "Sets autorepair feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.GKEMachineType, "gke-machine-type", "", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.GKEMinNumOfNodes, "gke-min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEMaxNumOfNodes, "gke-max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEProjectId, "gke-project-id", "", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.GKEZone, "gke-zone", "", "", "The compute zone (e.g. us-central1-a) for the cluster")

	// install options
	options.InstallOptions.addInstallFlags(cmd, true)
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
	err := o.installRequirements(GKE, "terraform", o.InstallOptions.InitOptions.HelmBinary())
	if err != nil {
		return err
	}

	if len(o.Flags.Cluster) >= 1 {
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

	err = o.createOrganisationGitRepo()
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

	jxEnvironment := ""

	for i := 1; i <= numOfClusters; i++ {
		var name string
		var provider string

		defaultOption := ""
		if i == 1 {
			defaultOption = "dev"
		}
		prompts := &survey.Input{
			Message: fmt.Sprintf("Cluster %v name:", i),
			Default: defaultOption,
		}
		validator := survey.Required
		err := survey.AskOne(prompts, &name, validator)
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
		err = survey.AskOne(prompts, &provider, validator)
		if err != nil {
			return err
		}

		if jxEnvironment == "" {
			confirm := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Would you like to install Jenkins X in cluster %v", name),
				Default: true,
			}
			survey.AskOne(prompt, &confirm, nil)

			if confirm {
				jxEnvironment = name
			}
		}
		c := GKECluster{_Name: name, _Provider: provider}

		o.Clusters = append(o.Clusters, c)
	}

	o.Flags.JxEnvironment = jxEnvironment

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

		c := GKECluster{_Name: pair[0], _Provider: pair[1]}
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

	if o.Flags.OrganisationName == "" {
		o.Flags.OrganisationName = strings.ToLower(randomdata.SillyName())
	}

	defaultRepoName := fmt.Sprintf("organisation-%s", o.Flags.OrganisationName)

	var dir string

	if o.Flags.LocalOrganisationRepository != "" {
		exists, err := util.FileExists(o.Flags.LocalOrganisationRepository)
		if err != nil {
			return err
		}
		if exists {
			dir = o.Flags.LocalOrganisationRepository
		} else {
			return errors.New("unable to find local repository " + o.Flags.LocalOrganisationRepository)
		}
	} else {
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
			fmt.Fprintf(o.Stdout(), "Remote repository %s\n\n", util.ColorInfo(repo.HTMLURL))
		}
	}

	// create directory structure
	clusterDefinitions, err := o.createOrganisationFolderStructure(dir)
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

	fmt.Fprintf(o.Stdout(), "Pushed git repository %s\n", util.ColorInfo(dir))

	fmt.Fprintf(o.Stdout(), "Creating Clusters...\n")
	err = o.createClusters(dir, clusterDefinitions)
	if err != nil {
		return err
	}

	if !o.Flags.SkipTerraformApply {
		devCluster, err := o.findDevCluster(clusterDefinitions)
		if err != nil {
			fmt.Fprintf(o.Stdout(), "Skipping jx install\n")
		} else {
			err = o.installJx(devCluster, clusterDefinitions)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(o.Stdout(), "Skipping jx install\n")
	}

	return nil
}

func (o *CreateTerraformOptions) createOrganisationFolderStructure(dir string) ([]Cluster, error) {
	o.writeGitIgnoreFile(dir)

	clusterDefinitions := []Cluster{}

	for _, c := range o.Clusters {
		fmt.Fprintf(o.Stdout(), "Creating config for cluster %s\n\n", util.ColorInfo(c.Name()))

		path := filepath.Join(dir, Clusters, c.Name(), Terraform)
		exists, err := util.FileExists(path)
		if err != nil {
			return nil, fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
		}

		if !exists {
			os.MkdirAll(path, DefaultWritePermissions)

			switch c.Provider() {
			case "gke":
				o.Git().Clone(TerraformTemplatesGKE, path)
				g := c.(GKECluster)

				err := o.configureGKECluster(&g, path)
				if err != nil {
					return nil, err
				}
				clusterDefinitions = append(clusterDefinitions, g)

			case "aks":
				// TODO add aks terraform templates URL
				return nil, fmt.Errorf("creating an AKS cluster via terraform is not currently supported")
			case "eks":
				// TODO add eks terraform templates URL
				return nil, fmt.Errorf("creating an EKS cluster via terraform is not currently supported")
			default:
				return nil, fmt.Errorf("unknown kubernetes provider type %s must be one of %v", c.Provider(), validTerraformClusterProviders)
			}
			os.RemoveAll(filepath.Join(path, ".git"))
			os.RemoveAll(filepath.Join(path, ".gitignore"))
		} else {
			// if the directory already exists, try to load its config

			switch c.Provider() {
			case "gke":
				g := c.(GKECluster)
				terraformVars := filepath.Join(path, "terraform.tfvars")
				fmt.Fprintf(o.Stdout(), "loading config from %s\n", util.ColorInfo(terraformVars))

				g.ParseTfVarsFile(terraformVars)
				clusterDefinitions = append(clusterDefinitions, g)

			case "aks":
				// TODO add aks terraform templates URL
				return nil, fmt.Errorf("creating an AKS cluster via terraform is not currently supported")
			case "eks":
				// TODO add eks terraform templates URL
				return nil, fmt.Errorf("creating an EKS cluster via terraform is not currently supported")
			default:
				return nil, fmt.Errorf("unknown kubernetes provider type %s must be one of %v", c.Provider(), validTerraformClusterProviders)
			}
		}
	}

	return clusterDefinitions, nil
}

func (o *CreateTerraformOptions) createClusters(dir string, clusterDefinitions []Cluster) error {
	fmt.Printf("Creating/Updating %v clusters\n", util.ColorInfo(len(clusterDefinitions)))
	for _, c := range clusterDefinitions {
		switch v := c.(type) {
		case GKECluster:
			path := filepath.Join(dir, Clusters, v.Name(), Terraform)
			fmt.Fprintf(o.Stdout(), "\n\nCreating/Updating cluster %s\n", util.ColorInfo(c.Name()))
			err := o.applyTerraformGKE(&v, path)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown kubernetes provider type, must be one of %v, got %s", validTerraformClusterProviders, v)
		}
	}

	return nil
}

func (o *CreateTerraformOptions) findDevCluster(clusterDefinitions []Cluster) (Cluster, error) {
	for _, c := range clusterDefinitions {
		if c.Name() == o.Flags.JxEnvironment {
			return c, nil
		}
	}
	return nil, fmt.Errorf("Unable to find jx environment %s", o.Flags.JxEnvironment)
}

func (o *CreateTerraformOptions) writeGitIgnoreFile(dir string) error {
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		file, err := os.OpenFile(gitignore, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString("**/*.key.json\n.terraform\n**/*.tfstate\n")
		if err != nil {
			return err
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

func (o *CreateTerraformOptions) configureGKECluster(g *GKECluster, path string) error {
	g.DiskSize = o.Flags.GKEDiskSize
	g.AutoUpgrade = o.Flags.GKEAutoUpgrade
	g.AutoRepair = o.Flags.GKEAutoRepair
	g.MachineType = o.Flags.GKEMachineType
	g.Zone = o.Flags.GKEZone
	g.ProjectId = o.Flags.GKEProjectId
	g.MinNumOfNodes = o.Flags.GKEMinNumOfNodes
	g.MaxNumOfNodes = o.Flags.GKEMaxNumOfNodes
	g.ServiceAccount = o.Flags.GKEServiceAccount
	g.Organisation = o.Flags.OrganisationName

	if g.ServiceAccount != "" {
		err := gke.Login(g.ServiceAccount, false)
		if err != nil {
			return err
		}
	}

	if g.ProjectId == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		g.ProjectId = projectId
	}

	if !o.Flags.GKESkipEnableApis {
		err := gke.EnableApis(g.ProjectId, "iam", "compute", "container")
		if err != nil {
			return err
		}
	}

	if g.Name() == "" {
		g._Name = strings.ToLower(randomdata.SillyName())
		fmt.Fprintf(o.Stdout(), "No cluster name provided so using a generated one: %s\n", util.ColorInfo(g.Name()))
	}

	if g.Zone == "" {
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

		err = survey.AskOne(prompts, &g.Zone, nil)
		if err != nil {
			return err
		}
	}

	if g.MachineType == "" {
		prompts := &survey.Select{
			Message:  "Google Cloud Machine Type:",
			Options:  gke.GetGoogleMachineTypes(),
			Help:     "We recommend a minimum of n1-standard-2 for Jenkins X,  a table of machine descriptions can be found here https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture",
			PageSize: 10,
			Default:  "n1-standard-2",
		}

		err := survey.AskOne(prompts, &g.MachineType, nil)
		if err != nil {
			return err
		}
	}

	if g.MinNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Minimum number of Nodes",
			Default: "3",
			Help:    "We recommend a minimum of 3 for Jenkins X,  the minimum number of nodes to be created in each of the cluster's zones",
		}

		err := survey.AskOne(prompt, &g.MinNumOfNodes, nil)
		if err != nil {
			return err
		}
	}

	if g.MaxNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Maximum number of Nodes",
			Default: "5",
			Help:    "We recommend at least 5 for Jenkins X,  the maximum number of nodes to be created in each of the cluster's zones",
		}

		err := survey.AskOne(prompt, &g.MaxNumOfNodes, nil)
		if err != nil {
			return err
		}
	}

	terraformVars := filepath.Join(path, "terraform.tfvars")
	err := g.CreateTfVarsFile(terraformVars)
	if err != nil {
		return err
	}

	storageBucket := fmt.Sprintf(gkeBucketConfiguration, validTerraformVersions, g.ProjectId, o.Flags.OrganisationName, g.Name())
	o.Debugf("Using bucket configuration %s", storageBucket)

	terraformTf := filepath.Join(path, "terraform.tf")
	// file exists
	if _, err := os.Stat(terraformTf); os.IsNotExist(err) {
		file, err := os.OpenFile(terraformTf, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString(storageBucket)
		if err != nil {
			return err
		}

		log.Infof("Created %s\n", terraformTf)
	}

	return nil
}

func (o *CreateTerraformOptions) applyTerraformGKE(g *GKECluster, path string) error {
	if g.ProjectId == "" {
		return errors.New("Unable to apply terraform, projectId has not been set")
	}

	log.Info("Applying Terraform changes\n")
	user, err := os_user.Current()
	if err != nil {
		return err
	}

	terraformVars := filepath.Join(path, "terraform.tfvars")

	if g.ServiceAccount == "" {
		if o.Flags.GKEServiceAccount != "" {
			g.ServiceAccount = o.Flags.GKEServiceAccount
			err := gke.Login(g.ServiceAccount, false)
			if err != nil {
				return err
			}

			err = gke.EnableApis(g.ProjectId, "iam", "compute", "container")
			if err != nil {
				return err
			}
		}
	}

	var serviceAccountPath string
	if g.ServiceAccount == "" {
		serviceAccountName := fmt.Sprintf("jx-%s-%s", o.Flags.OrganisationName, g.Name())
		fmt.Fprintf(o.Stdout(), "No GCP service account provided, creating %s\n", util.ColorInfo(serviceAccountName))

		_, err = gke.GetOrCreateServiceAccount(serviceAccountName, g.ProjectId, filepath.Dir(path))
		if err != nil {
			return err
		}
		serviceAccountPath = filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.key.json", serviceAccountName))
		fmt.Fprintf(o.Stdout(), "Created GCP service account: %s\n", util.ColorInfo(serviceAccountPath))
	} else {
		serviceAccountPath = g.ServiceAccount
		fmt.Fprintf(o.Stdout(), "Using provided GCP service account: %s\n", util.ColorInfo(serviceAccountPath))
	}

	// create the bucket
	bucketName := fmt.Sprintf("%s-%s-terraform-state", g.ProjectId, o.Flags.OrganisationName)
	exists, err := gke.BucketExists(g.ProjectId, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = gke.CreateBucket(g.ProjectId, bucketName, g.Region())
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Stdout(), "Created GCS bucket: %s in region %s\n", util.ColorInfo(bucketName), util.ColorInfo(g.Region()))
	}

	err = terraform.Init(path, serviceAccountPath)
	if err != nil {
		return err
	}

	plan, err := terraform.Plan(path, terraformVars, serviceAccountPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(o.Stdout(), plan)

	if !o.BatchMode {
		confirm := false
		prompt := &survey.Confirm{
			Message: "Would you like to apply this plan?",
		}
		survey.AskOne(prompt, &confirm, nil)

		if !confirm {
			// exit at this point
			return nil
		}
	}

	if !o.Flags.IgnoreTerraformWarnings {
		if strings.Contains(plan, "forces new resource") {
			fmt.Fprintf(o.Stdout(), "%s\n", util.ColorError("It looks like this plan is destructive, aborting."))
			fmt.Fprintf(o.Stdout(), "Use --ignore-terraform-warnings to override\n")
			return errors.New("Aborting destructive plan")
		}
	}

	if !o.Flags.SkipTerraformApply {
		log.Info("Applying plan...\n")

		err = terraform.Apply(path, terraformVars, serviceAccountPath, o.Out, o.Err)
		if err != nil {
			return err
		}

		// should we setup the labels at this point?
		args := []string{"container",
			"clusters",
			"update",
			g.ClusterName(),
			"--project",
			g.ProjectId,
			"--zone",
			g.Zone}

		labels := ""
		if err == nil && user != nil {
			username := sanitizeLabel(user.Username)
			if username != "" {
				sep := ""
				if labels != "" {
					sep = ","
				}
				labels += sep + "created-by=" + username
			}
		}

		sep := ""
		if labels != "" {
			sep = ","
		}
		labels += sep + fmt.Sprintf("created-with=terraform,created-on=%s", time.Now().Format("20060102150405"))
		args = append(args, "--update-labels="+strings.ToLower(labels))

		err = o.runCommand("gcloud", args...)
		if err != nil {
			return err
		}

		output, err := o.getCommandOutput("", "gcloud", "container", "clusters", "get-credentials", g.ClusterName(), "--zone", g.Zone, "--project", g.ProjectId)
		if err != nil {
			return err
		}
		log.Info(output)
	} else {
		fmt.Fprintf(o.Stdout(), "Skipping terraform apply\n")
	}
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

func (o *CreateTerraformOptions) installJx(c Cluster, clusters []Cluster) error {
	log.Infof("\n\nInstalling jx on cluster %s with context %s\n", util.ColorInfo(c.Name()), util.ColorInfo(c.Context()))

	err := o.runCommand("kubectl", "config", "use-context", c.Context())
	if err != nil {
		return err
	}

	// check if jx is already installed
	_, err = o.findEnvironmentNamespace(c.Name())
	if err != nil {
		// jx is missing, install,
		o.InstallOptions.Flags.DefaultEnvironmentPrefix = c.ClusterName()
		err = o.initAndInstall(c.Provider())
		if err != nil {
			return err
		}

		context, err := o.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return err
		}

		ns := o.InstallOptions.Flags.Namespace
		if ns == "" {
			_, ns, _ = o.KubeClient()
			if err != nil {
				return err
			}
		}
		err = o.runCommand("kubectl", "config", "set-context", context, "--namespace", ns)
		if err != nil {
			return err
		}

		err = o.runCommand("kubectl", "get", "ingress")
		if err != nil {
			return err
		}

		// if more than 1 clusters are defined, we will install an environment in each
		if len(clusters) > 1 {
			err = o.configureEnvironments(clusters)
			if err != nil {
				return err
			}
		}

		return err
	} else {
		log.Info("Skipping installing jx as it appears to be already installed\n")
	}

	return nil
}

func (o *CreateTerraformOptions) initAndInstall(provider string) error {
	// call jx init
	o.InstallOptions.BatchMode = o.BatchMode
	o.InstallOptions.Flags.Provider = provider

	if len(o.Clusters) > 1 {
		o.InstallOptions.Flags.NoDefaultEnvironments = true
		log.Info("Creating custom environments in each cluster\n")
	} else {
		log.Info("Creating default environments\n")
	}

	// call jx install
	installOpts := &o.InstallOptions

	err := installOpts.Run()
	if err != nil {
		return err
	}

	return nil
}

func (o *CreateTerraformOptions) configureEnvironments(clusters []Cluster) error {

	for index, cluster := range clusters {
		if cluster.Name() != o.Flags.JxEnvironment {
			jxClient, _, err := o.JXClient()
			if err != nil {
				return err
			}

			log.Infof("Checking for environments %s on cluster %s\n", cluster.Name(), cluster.ClusterName())
			_, envNames, err := kube.GetEnvironments(jxClient, cluster.Name())

			if err != nil || len(envNames) <= 1 {
				environmentOrder := (index) * 100
				o.InstallOptions.CreateEnvOptions.Options.Name = cluster.Name()
				o.InstallOptions.CreateEnvOptions.Options.Spec.Label = cluster.Name()
				o.InstallOptions.CreateEnvOptions.Options.Spec.Order = int32(environmentOrder)
				o.InstallOptions.CreateEnvOptions.GitRepositoryOptions.Owner = o.InstallOptions.Flags.EnvironmentGitOwner
				o.InstallOptions.CreateEnvOptions.Prefix = cluster.ClusterName()
				o.InstallOptions.CreateEnvOptions.Options.ClusterName = cluster.ClusterName()
				if o.BatchMode {
					o.InstallOptions.CreateEnvOptions.BatchMode = o.BatchMode
				}

				err := o.InstallOptions.CreateEnvOptions.Run()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
