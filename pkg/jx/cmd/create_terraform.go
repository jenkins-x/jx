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

	"time"

	"path"

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
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// Cluster interface for Clusters
type Cluster interface {
	Name() string
	SetName(string) string
	ClusterName() string
	Provider() string
	SetProvider(string) string
	Context() string
	CreateTfVarsFile(path string) error
}

// GKECluster implements Cluster interface for GKE
type GKECluster struct {
	name           string
	provider       string
	Organisation   string
	ProjectID      string
	Zone           string
	MachineType    string
	MinNumOfNodes  string
	MaxNumOfNodes  string
	DiskSize       string
	AutoRepair     bool
	AutoUpgrade    bool
	ServiceAccount string
}

// Name Get name
func (g GKECluster) Name() string {
	return g.name
}

// SetName Sets the name
func (g *GKECluster) SetName(name string) string {
	g.name = name
	return g.name
}

// ClusterName get cluster name
func (g GKECluster) ClusterName() string {
	return fmt.Sprintf("%s-%s", g.Organisation, g.name)
}

// Provider get provider
func (g GKECluster) Provider() string {
	return g.provider
}

// SetProvider Set the provider
func (g *GKECluster) SetProvider(provider string) string {
	g.provider = provider
	return g.provider
}

// Context Get the context
func (g GKECluster) Context() string {
	return fmt.Sprintf("%s_%s_%s_%s", "gke", g.ProjectID, g.Zone, g.ClusterName())
}

// Region Get the region
func (g GKECluster) Region() string {
	return gke.GetRegionFromZone(g.Zone)
}

// CreateTfVarsFile create vars
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
	err = terraform.WriteKeyValueToFileIfNotExists(path, "provider", g.provider)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "gcp_zone", g.Zone)
	if err != nil {
		return err
	}
	err = terraform.WriteKeyValueToFileIfNotExists(path, "gcp_project", g.ProjectID)
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

// ParseTfVarsFile Parse vars file
func (g *GKECluster) ParseTfVarsFile(path string) {
	g.Zone, _ = terraform.ReadValueFromFile(path, "gcp_zone")
	g.Organisation, _ = terraform.ReadValueFromFile(path, "organisation")
	g.provider, _ = terraform.ReadValueFromFile(path, "provider")
	g.ProjectID, _ = terraform.ReadValueFromFile(path, "gcp_project")
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

// Flags for a cluster
type Flags struct {
	Cluster                     []string
	OrganisationName            string
	SkipLogin                   bool
	ForkOrganisationGitRepo     string
	SkipTerraformApply          bool
	IgnoreTerraformWarnings     bool
	JxEnvironment               string
	GKEProjectID                string
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
	validTerraformClusterProviders = []string{"gke", "jx-infra"}

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
	// Clusters constant
	Clusters = "clusters"
	// Terraform constant
	Terraform = "terraform"
	// TerraformTemplatesGKE constant
	TerraformTemplatesGKE = "https://github.com/jenkins-x/terraform-jx-templates-gke.git"
)

// NewCmdCreateTerraform creates a command object for the "create" command
func NewCmdCreateTerraform(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateTerraformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
		InstallOptions: CreateInstallOptions(f, in, out, errOut),
	}

	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Creates a Jenkins X Terraform plan",
		Example: createTerraformExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.InstallOptions.addInstallFlags(cmd, true)
	options.addCommonFlags(cmd)
	options.addFlags(cmd, true)

	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")
	cmd.Flags().StringVarP(&options.Flags.GKEServiceAccount, "gke-service-account", "", "", "The service account to use to connect to GKE")
	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation Git repos")

	return cmd
}

func (options *CreateTerraformOptions) addFlags(cmd *cobra.Command, addSharedFlags bool) {
	// global flags
	if addSharedFlags {
		cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	}
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, "cluster", "c", []string{}, "Name and Kubernetes provider (gke, aks, eks) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().BoolVarP(&options.Flags.SkipTerraformApply, "skip-terraform-apply", "", false, "Skip applying the generated Terraform plans")
	cmd.Flags().BoolVarP(&options.Flags.IgnoreTerraformWarnings, "ignore-terraform-warnings", "", false, "Ignore any warnings about the Terraform plan being potentially destructive")
	cmd.Flags().StringVarP(&options.Flags.JxEnvironment, "jx-environment", "", "dev", "The cluster name to install jx inside")
	cmd.Flags().StringVarP(&options.Flags.LocalOrganisationRepository, "local-organisation-repository", "", "", "Rather than cloning from a remote Git server, the local directory to use for the organisational folder")

	// GKE specific overrides
	cmd.Flags().StringVarP(&options.Flags.GKEDiskSize, "gke-disk-size", "", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoUpgrade, "gke-enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoRepair, "gke-enable-autorepair", "", true, "Sets autorepair feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.GKEMachineType, "gke-machine-type", "", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.GKEMinNumOfNodes, "gke-min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEMaxNumOfNodes, "gke-max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEProjectID, "gke-project-id", "", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.GKEZone, "gke-zone", "", "", "The compute zone (e.g. us-central1-a) for the cluster")

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
func (options *CreateTerraformOptions) Run() error {
	err := options.installRequirements(GKE, "terraform", options.InstallOptions.InitOptions.HelmBinary())
	if err != nil {
		return err
	}

	if !options.Flags.SkipLogin {
		err = options.runCommandVerbose("gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}

	options.InstallOptions.Flags.Prow = true

	err = terraform.CheckVersion()
	if err != nil {
		return err
	}

	err = options.InstallOptions.InitOptions.validateGit()
	if err != nil {
		return err
	}

	if len(options.Flags.Cluster) >= 1 {
		err := options.ValidateClusterDetails()
		if err != nil {
			return err
		}
	}

	if len(options.Flags.Cluster) == 0 {
		err := options.ClusterDetailsWizard()
		if err != nil {
			return err
		}
	}

	err = options.createOrganisationGitRepo()
	if err != nil {
		return err
	}
	return nil
}

// ClusterDetailsWizard cluster details wizard
func (options *CreateTerraformOptions) ClusterDetailsWizard() error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	var numOfClusters int
	numOfClustersStr := "1"
	prompts := &survey.Input{
		Message: fmt.Sprintf("How many clusters shall we create?"),
		Default: numOfClustersStr,
	}

	err := survey.AskOne(prompts, &numOfClustersStr, nil, surveyOpts)
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
		err := survey.AskOne(prompts, &name, validator, surveyOpts)
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
					return fmt.Errorf("invalid cluster provider type %s, must be one of %v", str, validTerraformClusterProviders)
				}
				return nil
			},
		)
		err = survey.AskOne(prompts, &provider, validator, surveyOpts)
		if err != nil {
			return err
		}

		if jxEnvironment == "" {
			confirm := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Would you like to install Jenkins X in cluster %v", name),
				Default: true,
			}
			survey.AskOne(prompt, &confirm, nil, surveyOpts)

			if confirm {
				jxEnvironment = name
			}
		}
		c := &GKECluster{name: name, provider: provider}

		options.Clusters = append(options.Clusters, c)
	}

	options.Flags.JxEnvironment = jxEnvironment

	return nil
}

// ValidateClusterDetails validates the options for a cluster
func (options *CreateTerraformOptions) ValidateClusterDetails() error {
	for _, p := range options.Flags.Cluster {
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			return errors.New("need to provide cluster values as --cluster name=provider, e.g. --cluster production=gke")
		}
		if !stringInValidProviders(pair[1]) {
			return fmt.Errorf("invalid cluster provider type %s, must be one of %v", p, validTerraformClusterProviders)
		}

		c := &GKECluster{name: pair[0], provider: pair[1]}
		options.Clusters = append(options.Clusters, c)
	}
	return nil
}

func (options *CreateTerraformOptions) createOrganisationGitRepo() error {
	organisationDir, err := util.OrganisationsDir()
	if err != nil {
		return err
	}

	authConfigSvc, err := options.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	if options.Flags.OrganisationName == "" {
		options.Flags.OrganisationName = strings.ToLower(randomdata.SillyName())
	}

	defaultRepoName := fmt.Sprintf("organisation-%s", options.Flags.OrganisationName)

	var dir string

	if options.Flags.LocalOrganisationRepository != "" {
		exists, err := util.FileExists(options.Flags.LocalOrganisationRepository)
		if err != nil {
			return err
		}
		if exists {
			dir = options.Flags.LocalOrganisationRepository
		} else {
			return errors.New("unable to find local repository " + options.Flags.LocalOrganisationRepository)
		}
	} else {
		details, err := gits.PickNewOrExistingGitRepository(options.BatchMode, authConfigSvc,
			defaultRepoName, &options.GitRepositoryOptions, nil, nil, options.Git(), true, options.In, options.Out, options.Err)
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
			fmt.Fprintf(options.Out, "Creating Git repository %s/%s\n", util.ColorInfo(owner), util.ColorInfo(repoName))

			repo, err = details.CreateRepository()
			if err != nil {
				return err
			}

			dir, err = util.CreateUniqueDirectory(organisationDir, details.RepoName, util.MaximumNewDirectoryAttempts)
			if err != nil {
				return err
			}

			err = options.Git().Clone(options.Flags.ForkOrganisationGitRepo, dir)
			if err != nil {
				return err
			}
			pushGitURL, err := options.Git().CreatePushURL(repo.CloneURL, details.User)
			if err != nil {
				return err
			}
			err = options.Git().AddRemote(dir, "upstream", options.Flags.ForkOrganisationGitRepo)
			if err != nil {
				return err
			}
			err = options.Git().SetRemoteURL(dir, "origin", pushGitURL)
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(options.Out, "Git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))

			dir = path.Join(organisationDir, details.RepoName)
			localDirExists, err := util.FileExists(dir)
			if err != nil {
				return err
			}

			if localDirExists {
				// if remote repo does exist & local does exist, git pull the local repo
				fmt.Fprintf(options.Out, "local directory already exists\n")

				err = options.Git().Pull(dir)
				if err != nil {
					return err
				}
			} else {
				fmt.Fprintf(options.Out, "cloning repository locally\n")
				err = os.MkdirAll(dir, os.FileMode(0755))
				if err != nil {
					return err
				}

				// if remote repo does exist & local directory does not exist, clone locally
				pushGitURL, err := options.Git().CreatePushURL(repo.CloneURL, details.User)
				if err != nil {
					return err
				}
				err = options.Git().Clone(pushGitURL, dir)
				if err != nil {
					return err
				}
			}
			fmt.Fprintf(options.Out, "Remote repository %s\n\n", util.ColorInfo(repo.HTMLURL))
		}

		options.InstallOptions.Flags.EnvironmentGitOwner = org
	}

	// create directory structure
	clusterDefinitions, err := options.CreateOrganisationFolderStructure(dir)
	if err != nil {
		return err
	}

	changes, err := options.commitClusters(dir)
	if err != nil {
		return err
	}

	if changes {
		err = options.Git().PushMaster(dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(options.Out, "Pushed Git repository %s\n", util.ColorInfo(dir))
	}

	fmt.Fprintf(options.Out, "Creating Clusters...\n")
	err = options.createClusters(dir, clusterDefinitions)
	if err != nil {
		return err
	}

	if !options.Flags.SkipTerraformApply {
		devCluster, err := options.findDevCluster(clusterDefinitions)
		if err != nil {
			fmt.Fprintf(options.Out, "Skipping jx install\n")
		} else {
			err = options.installJx(devCluster, clusterDefinitions)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(options.Out, "Skipping jx install\n")
	}

	return nil
}

// CreateOrganisationFolderStructure creates an organisations folder structure
func (options *CreateTerraformOptions) CreateOrganisationFolderStructure(dir string) ([]Cluster, error) {
	options.writeGitIgnoreFile(dir)

	clusterDefinitions := []Cluster{}

	for _, c := range options.Clusters {
		fmt.Fprintf(options.Out, "Creating config for cluster %s\n\n", util.ColorInfo(c.Name()))

		path := filepath.Join(dir, Clusters, c.Name(), Terraform)
		exists, err := util.FileExists(path)
		if err != nil {
			return nil, fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
		}

		if !exists {
			options.Debugf("cluster %s does not exist, creating...", c.Name())

			os.MkdirAll(path, DefaultWritePermissions)

			switch c.Provider() {
			case "gke", "jx-infra":
				options.Git().Clone(TerraformTemplatesGKE, path)
				g := c.(*GKECluster)
				//g := &GKECluster{}

				err := options.configureGKECluster(g, path)
				if err != nil {
					return nil, err
				}
				clusterDefinitions = append(clusterDefinitions, g)

			case "aks":
				// TODO add aks terraform templates URL
				return nil, fmt.Errorf("creating an AKS cluster via Terraform is not currently supported")
			case "eks":
				// TODO add eks terraform templates URL
				return nil, fmt.Errorf("creating an EKS cluster via Terraform is not currently supported")
			default:
				return nil, fmt.Errorf("unknown Kubernetes provider type %s must be one of %v", c.Provider(), validTerraformClusterProviders)
			}
			os.RemoveAll(filepath.Join(path, ".git"))
			os.RemoveAll(filepath.Join(path, ".gitignore"))
		} else {
			// if the directory already exists, try to load its config
			options.Debugf("cluster %s already exists, loading...", c.Name())

			switch c.Provider() {
			case "gke", "jx-infra":
				//g := &GKECluster{}
				g := c.(*GKECluster)
				terraformVars := filepath.Join(path, "terraform.tfvars")
				fmt.Fprintf(options.Out, "loading config from %s\n", util.ColorInfo(terraformVars))

				g.ParseTfVarsFile(terraformVars)
				clusterDefinitions = append(clusterDefinitions, g)

			case "aks":
				// TODO add aks terraform templates URL
				return nil, fmt.Errorf("creating an AKS cluster via Terraform is not currently supported")
			case "eks":
				// TODO add eks terraform templates URL
				return nil, fmt.Errorf("creating an EKS cluster via Terraform is not currently supported")
			default:
				return nil, fmt.Errorf("unknown Kubernetes provider type %s must be one of %v", c.Provider(), validTerraformClusterProviders)
			}
		}
	}

	return clusterDefinitions, nil
}

func (options *CreateTerraformOptions) createClusters(dir string, clusterDefinitions []Cluster) error {
	fmt.Printf("Creating/Updating %v clusters\n", util.ColorInfo(len(clusterDefinitions)))
	for _, c := range clusterDefinitions {
		switch v := c.(type) {
		case *GKECluster:
			path := filepath.Join(dir, Clusters, v.Name(), Terraform)
			fmt.Fprintf(options.Out, "\n\nCreating/Updating cluster %s\n", util.ColorInfo(c.Name()))
			err := options.applyTerraformGKE(v, path)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown Kubernetes provider type, must be one of %v, got %s", validTerraformClusterProviders, v)
		}
	}

	return nil
}

func (options *CreateTerraformOptions) findDevCluster(clusterDefinitions []Cluster) (Cluster, error) {
	for _, c := range clusterDefinitions {
		if c.Name() == options.Flags.JxEnvironment {
			return c, nil
		}
	}
	return nil, fmt.Errorf("Unable to find jx environment %s", options.Flags.JxEnvironment)
}

func (options *CreateTerraformOptions) writeGitIgnoreFile(dir string) error {
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		file, err := os.OpenFile(gitignore, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString("**/*.key.json\n.terraform\n**/*.tfstate\njx\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *CreateTerraformOptions) commitClusters(dir string) (bool, error) {
	err := options.Git().Add(dir, "*")
	if err != nil {
		return false, err
	}
	changes, err := options.Git().HasChanges(dir)
	if err != nil {
		return false, err
	}
	if changes {
		return true, options.Git().CommitDir(dir, "Add organisation clusters")
	}
	return false, nil
}

func (options *CreateTerraformOptions) configureGKECluster(g *GKECluster, path string) error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	g.DiskSize = options.Flags.GKEDiskSize
	g.AutoUpgrade = options.Flags.GKEAutoUpgrade
	g.AutoRepair = options.Flags.GKEAutoRepair
	g.MachineType = options.Flags.GKEMachineType
	g.Zone = options.Flags.GKEZone
	g.ProjectID = options.Flags.GKEProjectID
	g.MinNumOfNodes = options.Flags.GKEMinNumOfNodes
	g.MaxNumOfNodes = options.Flags.GKEMaxNumOfNodes
	g.ServiceAccount = options.Flags.GKEServiceAccount
	g.Organisation = options.Flags.OrganisationName

	if g.ServiceAccount != "" {
		options.Debugf("loading service account for cluster %s", g.Name())

		err := gke.Login(g.ServiceAccount, false)
		if err != nil {
			return err
		}
	}

	if g.ProjectID == "" {
		options.Debugf("determining google project for cluster %s", g.Name())

		projectID, err := options.getGoogleProjectID()
		if err != nil {
			return err
		}
		g.ProjectID = projectID
	}

	if !options.Flags.GKESkipEnableApis {
		options.Debugf("enabling apis for %s", g.Name())

		err := gke.EnableAPIs(g.ProjectID, "iam", "compute", "container")
		if err != nil {
			return err
		}
	}

	if g.Name() == "" {
		options.Debugf("generating a new name for cluster %s", g.Name())

		g.name = strings.ToLower(randomdata.SillyName())
		fmt.Fprintf(options.Out, "No cluster name provided so using a generated one: %s\n", util.ColorInfo(g.Name()))
	}

	if g.Zone == "" {
		options.Debugf("getting available zones for cluster %s", g.Name())

		availableZones, err := gke.GetGoogleZones(g.ProjectID)
		if err != nil {
			return err
		}
		prompts := &survey.Select{
			Message:  "Google Cloud Zone:",
			Options:  availableZones,
			PageSize: 10,
			Help:     "The compute zone (e.g. us-central1-a) for the cluster",
		}

		err = survey.AskOne(prompts, &g.Zone, nil, surveyOpts)
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

		err := survey.AskOne(prompts, &g.MachineType, nil, surveyOpts)
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

		err := survey.AskOne(prompt, &g.MinNumOfNodes, nil, surveyOpts)
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

		err := survey.AskOne(prompt, &g.MaxNumOfNodes, nil, surveyOpts)
		if err != nil {
			return err
		}
	}

	terraformVars := filepath.Join(path, "terraform.tfvars")
	err := g.CreateTfVarsFile(terraformVars)
	if err != nil {
		return err
	}

	storageBucket := fmt.Sprintf(gkeBucketConfiguration, validTerraformVersions, g.ProjectID, options.Flags.OrganisationName, g.Name())
	options.Debugf("Using bucket configuration %s", storageBucket)

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

func (options *CreateTerraformOptions) applyTerraformGKE(g *GKECluster, path string) error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	if g.ProjectID == "" {
		return errors.New("Unable to apply terraform, projectId has not been set")
	}

	log.Info("Applying Terraform changes\n")

	terraformVars := filepath.Join(path, "terraform.tfvars")

	if g.ServiceAccount == "" {
		if options.Flags.GKEServiceAccount != "" {
			g.ServiceAccount = options.Flags.GKEServiceAccount
			err := gke.Login(g.ServiceAccount, false)
			if err != nil {
				return err
			}

			options.Debugf("attempting to enable apis")
			err = gke.EnableAPIs(g.ProjectID, "iam", "compute", "container")
			if err != nil {
				return err
			}
		}
	}

	var serviceAccountPath string
	if g.ServiceAccount == "" {
		serviceAccountName := fmt.Sprintf("jx-%s-%s", options.Flags.OrganisationName, g.Name())
		fmt.Fprintf(options.Out, "No GCP service account provided, creating %s\n", util.ColorInfo(serviceAccountName))

		_, err := gke.GetOrCreateServiceAccount(serviceAccountName, g.ProjectID, filepath.Dir(path), gke.REQUIRED_SERVICE_ACCOUNT_ROLES)
		if err != nil {
			return err
		}
		serviceAccountPath = filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.key.json", serviceAccountName))
		fmt.Fprintf(options.Out, "Created GCP service account: %s\n", util.ColorInfo(serviceAccountPath))
	} else {
		serviceAccountPath = g.ServiceAccount
		fmt.Fprintf(options.Out, "Using provided GCP service account: %s\n", util.ColorInfo(serviceAccountPath))
	}

	// create the bucket
	bucketName := fmt.Sprintf("%s-%s-terraform-state", g.ProjectID, options.Flags.OrganisationName)
	exists, err := gke.BucketExists(g.ProjectID, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = gke.CreateBucket(g.ProjectID, bucketName, g.Region())
		if err != nil {
			return err
		}
		fmt.Fprintf(options.Out, "Created GCS bucket: %s in region %s\n", util.ColorInfo(bucketName), util.ColorInfo(g.Region()))
	}

	err = terraform.Init(path, serviceAccountPath)
	if err != nil {
		return err
	}

	plan, err := terraform.Plan(path, terraformVars, serviceAccountPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(options.Out, plan)

	if !options.BatchMode {
		confirm := false
		prompt := &survey.Confirm{
			Message: "Would you like to apply this plan?",
		}
		survey.AskOne(prompt, &confirm, nil, surveyOpts)

		if !confirm {
			// exit at this point
			return nil
		}
	}

	if !options.Flags.IgnoreTerraformWarnings {
		if strings.Contains(plan, "forces new resource") {
			fmt.Fprintf(options.Out, "%s\n", util.ColorError("It looks like this plan is destructive, aborting."))
			fmt.Fprintf(options.Out, "Use --ignore-terraform-warnings to override\n")
			return errors.New("aborting destructive plan")
		}
	}

	if !options.Flags.SkipTerraformApply {
		log.Info("Applying plan...\n")

		err = terraform.Apply(path, terraformVars, serviceAccountPath, options.Out, options.Err)
		if err != nil {
			return err
		}

		output, err := options.getCommandOutput("", "gcloud", "container", "clusters", "get-credentials", g.ClusterName(), "--zone", g.Zone, "--project", g.ProjectID)
		if err != nil {
			return err
		}
		log.Info(output)
	} else {
		fmt.Fprintf(options.Out, "Skipping Terraform apply\n")
	}
	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (options *CreateTerraformOptions) getGoogleProjectID() (string, error) {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	existingProjects, err := gke.GetGoogleProjects()
	if err != nil {
		return "", err
	}

	var projectID string
	if len(existingProjects) == 0 {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("No existing Google Projects exist, create one now?"),
			Default: true,
		}
		flag := true
		err = survey.AskOne(confirm, &flag, nil, surveyOpts)
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
		projectID = existingProjects[0]
		log.Infof("Using the only Google Cloud Project %s to create the cluster\n", util.ColorInfo(projectID))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		err := survey.AskOne(prompts, &projectID, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}

	if projectID == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectID, nil
}

func (options *CreateTerraformOptions) installJx(c Cluster, clusters []Cluster) error {
	log.Infof("\n\nInstalling jx on cluster %s with context %s\n", util.ColorInfo(c.Name()), util.ColorInfo(c.Context()))

	err := options.RunCommand("kubectl", "config", "use-context", c.Context())
	if err != nil {
		return err
	}

	// check if jx is already installed
	_, err = options.findEnvironmentNamespace(c.Name())
	if err != nil {
		// jx is missing, install,
		options.InstallOptions.Flags.DefaultEnvironmentPrefix = c.ClusterName()
		options.InstallOptions.Flags.Prow = true
		err = options.initAndInstall(c.Provider())
		if err != nil {
			return err
		}

		context, err := options.getCommandOutput("", "kubectl", "config", "current-context")
		if err != nil {
			return err
		}

		ns := options.InstallOptions.Flags.Namespace
		if ns == "" {
			_, ns, _ = options.KubeClientAndNamespace()
			if err != nil {
				return err
			}
		}
		err = options.RunCommand("kubectl", "config", "set-context", context, "--namespace", ns)
		if err != nil {
			return err
		}

		err = options.RunCommand("kubectl", "get", "ingress")
		if err != nil {
			return err
		}

		// if more than 1 clusters are defined, we will install an environment in each
		if len(clusters) > 1 {
			err = options.configureEnvironments(clusters)
			if err != nil {
				return err
			}
		}

		return err
	}
	log.Info("Skipping installing jx as it appears to be already installed\n")

	return nil
}

func (options *CreateTerraformOptions) initAndInstall(provider string) error {
	// call jx init
	options.InstallOptions.BatchMode = options.BatchMode
	options.InstallOptions.Flags.Provider = provider

	if len(options.Clusters) > 1 {
		options.InstallOptions.Flags.NoDefaultEnvironments = true
		log.Info("Creating custom environments in each cluster\n")
	} else {
		log.Info("Creating default environments\n")
	}

	// call jx install
	installOpts := &options.InstallOptions
	err := installOpts.Run()
	if err != nil {
		return err
	}

	return nil
}

func (options *CreateTerraformOptions) configureEnvironments(clusters []Cluster) error {

	for index, cluster := range clusters {
		if cluster.Name() != options.Flags.JxEnvironment {
			jxClient, _, err := options.JXClient()
			if err != nil {
				return err
			}
			log.Infof("Checking for environments %s on cluster %s\n", cluster.Name(), cluster.ClusterName())
			_, envNames, err := kube.GetEnvironments(jxClient, cluster.Name())

			if err != nil || len(envNames) <= 1 {
				environmentOrder := (index) * 100
				options.InstallOptions.CreateEnvOptions.Options.Name = cluster.Name()
				options.InstallOptions.CreateEnvOptions.Options.Spec.Label = cluster.Name()
				options.InstallOptions.CreateEnvOptions.Options.Spec.Order = int32(environmentOrder)
				options.InstallOptions.CreateEnvOptions.GitRepositoryOptions.Owner = options.InstallOptions.Flags.EnvironmentGitOwner
				options.InstallOptions.CreateEnvOptions.Prefix = cluster.ClusterName()
				options.InstallOptions.CreateEnvOptions.Options.ClusterName = cluster.ClusterName()
				if options.BatchMode {
					options.InstallOptions.CreateEnvOptions.BatchMode = options.BatchMode
				}

				err := options.InstallOptions.CreateEnvOptions.Run()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
