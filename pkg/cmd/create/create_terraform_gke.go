package create

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/features"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"fmt"

	os_user "os/user"

	"strconv"

	"errors"

	"path/filepath"

	"os"

	"time"

	"path"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/terraform"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
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
	Validate() error
}

// GKECluster implements Cluster interface for GKE
type GKECluster struct {
	name           string
	provider       string
	Organisation   string
	ProjectID      string
	Zone           string
	MachineType    string
	Preemptible    bool
	MinNumOfNodes  string
	MaxNumOfNodes  string
	DiskSize       string
	AutoRepair     bool
	AutoUpgrade    bool
	ServiceAccount string
	DevStorageRole string
	EnableKaniko   bool
	EnableVault    bool
}

const (
	devStorageFullControl = "https://www.googleapis.com/auth/devstorage.full_control"
	devStorageReadOnly    = "https://www.googleapis.com/auth/devstorage.read_only"
)

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

type terraformFileWriter struct {
	err error
}

func (tf *terraformFileWriter) write(path string, key string, value string) {
	if tf.err != nil {
		return
	}
	tf.err = terraform.WriteKeyValueToFileIfNotExists(path, key, value)
}

// Validate validates that all args are ok to create a GKE cluster
func (g GKECluster) Validate() error {
	if len(g.ClusterName()) >= 27 {
		return errors.New("cluster name must not be longer than 27 characters - " + g.ClusterName())
	}
	return nil
}

// CreateTfVarsFile create vars
func (g GKECluster) CreateTfVarsFile(path string) error {
	user, err := os_user.Current()
	var username string
	if err != nil {
		username = "unknown"
	} else {
		username = util.SanitizeLabel(user.Username)
	}

	tf := terraformFileWriter{}
	tf.write(path, "created_by", username)
	tf.write(path, "created_timestamp", time.Now().Format("Mon-Jan-2-2006-15:04:05"))
	tf.write(path, "cluster_name", g.ClusterName())
	tf.write(path, "organisation", g.Organisation)
	tf.write(path, "cloud_provider", g.provider)
	tf.write(path, "gcp_zone", g.Zone)
	tf.write(path, "gcp_region", g.Region())
	tf.write(path, "gcp_project", g.ProjectID)
	tf.write(path, "min_node_count", g.MinNumOfNodes)
	tf.write(path, "max_node_count", g.MaxNumOfNodes)
	tf.write(path, "node_machine_type", g.MachineType)
	tf.write(path, "node_preemptible", strconv.FormatBool(g.Preemptible))
	tf.write(path, "node_disk_size", g.DiskSize)
	tf.write(path, "auto_repair", strconv.FormatBool(g.AutoRepair))
	tf.write(path, "auto_upgrade", strconv.FormatBool(g.AutoUpgrade))
	tf.write(path, "enable_kubernetes_alpha", "false")
	tf.write(path, "enable_legacy_abac", "false")
	tf.write(path, "logging_service", "logging.googleapis.com")
	tf.write(path, "monitoring_service", "monitoring.googleapis.com")
	tf.write(path, "node_devstorage_role", g.DevStorageRole)
	tf.write(path, "enable_kaniko", booleanAsInt(g.EnableKaniko))
	tf.write(path, "enable_vault", booleanAsInt(g.EnableVault))

	if tf.err != nil {
		return err
	}

	return nil
}

// ParseTfVarsFile Parse vars file
func (g *GKECluster) ParseTfVarsFile(path string) {
	g.Zone, _ = terraform.ReadValueFromFile(path, "gcp_zone")
	// no need to read region as it can be calculated
	g.Organisation, _ = terraform.ReadValueFromFile(path, "organisation")
	g.provider, _ = terraform.ReadValueFromFile(path, "cloud_provider")
	g.ProjectID, _ = terraform.ReadValueFromFile(path, "gcp_project")
	g.MinNumOfNodes, _ = terraform.ReadValueFromFile(path, "min_node_count")
	g.MaxNumOfNodes, _ = terraform.ReadValueFromFile(path, "max_node_count")
	g.MachineType, _ = terraform.ReadValueFromFile(path, "node_machine_type")
	g.DiskSize, _ = terraform.ReadValueFromFile(path, "node_disk_size")
	g.DevStorageRole, _ = terraform.ReadValueFromFile(path, "node_devstorage_role")

	preemptible, _ := terraform.ReadValueFromFile(path, "node_preemptible")
	b, _ := strconv.ParseBool(preemptible)
	g.Preemptible = b

	autoRepair, _ := terraform.ReadValueFromFile(path, "auto_repair")
	b, _ = strconv.ParseBool(autoRepair)
	g.AutoRepair = b

	autoUpgrade, _ := terraform.ReadValueFromFile(path, "auto_upgrade")
	b, _ = strconv.ParseBool(autoUpgrade)
	g.AutoUpgrade = b

	enableKaniko, _ := terraform.ReadValueFromFile(path, "enable_kaniko")
	b, _ = strconv.ParseBool(enableKaniko)
	g.EnableKaniko = b

	enableVault, _ := terraform.ReadValueFromFile(path, "enable_vault")
	b, _ = strconv.ParseBool(enableVault)
	g.EnableVault = b
}

// TerraformGKEFlags for a cluster
type TerraformGKEFlags struct {
	OrganisationName            string
	ClusterName                 string
	SkipLogin                   bool
	ForkOrganisationGitRepo     string
	SkipTerraformApply          bool
	SkipInstallation            bool
	NoActiveCluster             bool
	IgnoreTerraformWarnings     bool
	JxEnvironment               string
	LocalOrganisationRepository string
}

// TerraformGKEOptions the options for the create spring command
type TerraformGKEOptions struct {
	CreateClusterGKEOptions
	Flags   Flags
	Cluster Cluster
}

var (
	validTerraformClusterProviders = []string{"gke", "jx-infra"}

	createTerraformGKEExample = templates.Examples(`
		jx create terraform gke

		# to specify the clusters via flags
		jx create terraform gke -o myorg -c dev 

`)

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
func NewCmdCreateTerraformGKE(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &TerraformGKEOptions{
		CreateClusterGKEOptions: CreateClusterGKEOptions{
			CreateClusterOptions: CreateClusterOptions{
				CreateOptions: CreateOptions{
					CommonOptions: commonOpts,
				},
				InstallOptions: CreateInstallOptions(commonOpts),
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Creates a Jenkins X Terraform plan for GKE",
		Example: createTerraformGKEExample,
		PreRun: func(cmd *cobra.Command, args []string) {
			err := features.IsEnabled(cmd)
			helper.CheckErr(err)

			err = options.InstallOptions.CheckFeatures()
			helper.CheckErr(err)

			options.InstallOptions.Flags.Provider = "gke"

			err = options.InstallOptions.CheckFlags()
			helper.CheckErr(err)
		},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.InstallOptions.AddInstallFlags(cmd, true)
	options.addFlags(cmd, true)

	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")
	cmd.Flags().StringVarP(&options.Flags.GKEServiceAccount, "gke-service-account", "", "", "The service account to use to connect to GKE")
	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation Git repos")

	return cmd
}

func (options *TerraformGKEOptions) addFlags(cmd *cobra.Command, addSharedFlags bool) {
	// global flags
	if addSharedFlags {
		cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	}
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, optionCluster, "c", []string{}, "Name and Kubernetes provider (currently gke only) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "", "", "The name of a single cluster to create - cannot be used in conjunction with --"+optionCluster)
	cmd.Flags().StringVarP(&options.Flags.CloudProvider, optionCloudProvider, "", "", "The cloud provider (currently gke only) - cannot be used in conjunction with --"+optionCluster)
	cmd.Flags().BoolVarP(&options.Flags.SkipTerraformApply, "skip-terraform-apply", "", false, "Skip applying the generated Terraform plans")
	cmd.Flags().BoolVarP(&options.Flags.SkipInstallation, "skip-installation", "", false, "Provision cluster only, don't install Jenkins X into it")
	cmd.Flags().BoolVarP(&options.Flags.IgnoreTerraformWarnings, "ignore-terraform-warnings", "", false, "Ignore any warnings about the Terraform plan being potentially destructive")
	cmd.Flags().StringVarP(&options.Flags.JxEnvironment, "jx-environment", "", "dev", "The cluster name to install jx inside")
	cmd.Flags().StringVarP(&options.Flags.LocalOrganisationRepository, "local-organisation-repository", "", "", "Rather than cloning from a remote Git server, the local directory to use for the organisational folder")
	cmd.Flags().BoolVarP(&options.Flags.NoActiveCluster, "no-active-cluster", "", false, "Tells JX there's isn't currently an active cluster, so we cannot use it for configuration")

	// GKE specific overrides
	cmd.Flags().StringVarP(&options.Flags.GKEDiskSize, "disk-size", "", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoUpgrade, "enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoRepair, "enable-autorepair", "", true, "Sets autorepair feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEPreemptible, "preemptible", "", false, "Use preemptible VMs in the node-pool")
	cmd.Flags().StringVarP(&options.Flags.GKEMachineType, "machine-type", "", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.GKEMinNumOfNodes, "min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEMaxNumOfNodes, "max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEProjectID, "project-id", "", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.GKEZone, "zone", "", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().BoolVarP(&options.Flags.GKEUseEnhancedScopes, "use-enhanced-scopes", "", false, "Use enhanced Oauth scopes for access to GCS/GCR")
	cmd.Flags().BoolVarP(&options.Flags.GKEUseEnhancedApis, "use-enhanced-apis", "", false, "Enable enhanced APIs to utilise Container Registry & Cloud Build")
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
func (options *TerraformGKEOptions) Run() error {
	err := options.InstallRequirements(cloud.GKE, "terraform", options.InstallOptions.InitOptions.HelmBinary())
	if err != nil {
		return err
	}

	if !options.Flags.SkipLogin {
		err = options.RunCommandVerbose("gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}

	log.Logger().Debugf("Checking terraform version")
	err = terraform.CheckVersion()
	if err != nil {
		return err
	}

	log.Logger().Debugf("Validating git configuration")
	err = options.InstallOptions.InitOptions.ValidateGit()
	if err != nil {
		return err
	}

	if options.Cluster == nil {
		log.Logger().Debugf("Launching cluster details wizard")
		err := options.ClusterDetailsWizard()
		if err != nil {
			return err
		}
	}

	if options.Flags.NoActiveCluster {
		err = options.SetFakeKubeClient()
		if err != nil {
			return err
		}
	}

	options.InstallOptions.Owner = options.InstallOptions.Flags.EnvironmentGitOwner
	log.Logger().Debugf("Creating organisation git repository")
	err = options.createOrganisationGitRepo()
	if err != nil {
		return err
	}
	return nil
}

// ClusterDetailsWizard cluster details wizard
func (options *TerraformGKEOptions) ClusterDetailsWizard() error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)

	jxEnvironment := ""

	var name string
	provider := cloud.GKE
	defaultOption := "dev"

	prompts := &survey.Input{
		Message: "Cluster name:",
		Default: defaultOption,
	}
	validator := survey.Required
	err := survey.AskOne(prompts, &name, validator, surveyOpts)
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
	options.Cluster = &GKECluster{name: name, provider: provider}
	options.Flags.JxEnvironment = jxEnvironment

	return nil
}

func (options *TerraformGKEOptions) createOrganisationGitRepo() error {
	organisationDir, err := util.OrganisationsDir()
	if err != nil {
		return err
	}

	authConfigSvc, err := options.GitAuthConfigService()
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
			defaultRepoName, &options.InstallOptions.GitRepositoryOptions, nil, nil, options.Git(), true, options.GetIOFileHandles())
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
			log.Logger().Infof("Creating Git repository %s/%s", util.ColorInfo(owner), util.ColorInfo(repoName))

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
			pushGitURL, err := options.Git().CreateAuthenticatedURL(repo.CloneURL, details.User)
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
			log.Logger().Infof("Git repository %s/%s already exists", util.ColorInfo(owner), util.ColorInfo(repoName))

			dir = path.Join(organisationDir, details.RepoName)
			localDirExists, err := util.FileExists(dir)
			if err != nil {
				return err
			}

			if localDirExists {
				// if remote repo does exist & local does exist, git pull the local repo
				log.Logger().Infof("local directory already exists")

				err = options.Git().Pull(dir)
				if err != nil {
					return err
				}
			} else {
				log.Logger().Infof("cloning repository locally")
				err = os.MkdirAll(dir, os.FileMode(0755))
				if err != nil {
					return err
				}

				// if remote repo does exist & local directory does not exist, clone locally
				pushGitURL, err := options.Git().CreateAuthenticatedURL(repo.CloneURL, details.User)
				if err != nil {
					return err
				}
				err = options.Git().Clone(pushGitURL, dir)
				if err != nil {
					return err
				}
			}
			log.Logger().Infof("Remote repository %s\n", util.ColorInfo(repo.HTMLURL))
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
		log.Logger().Infof("Pushed Git repository %s", util.ColorInfo(dir))
	}

	log.Logger().Infof("Creating Clusters...")
	err = options.createClusters(dir, clusterDefinitions)
	if err != nil {
		return err
	}

	if !options.Flags.SkipTerraformApply {
		devCluster, err := options.findDevCluster(clusterDefinitions)
		if err != nil {
			log.Logger().Infof("Skipping jx install")
		} else {
			if !options.Flags.SkipInstallation {
				err = options.installJx(devCluster, clusterDefinitions)
				if err != nil {
					return err
				}
			} else {
				log.Logger().Infof("Skipping jx install")
			}
		}
	} else {
		log.Logger().Infof("Skipping jx install")
	}

	return nil
}

// CreateOrganisationFolderStructure creates an organisations folder structure
func (options *TerraformGKEOptions) CreateOrganisationFolderStructure(dir string) ([]Cluster, error) {
	options.writeGitIgnoreFile(dir)

	clusterDefinitions := []Cluster{}

	c := options.Cluster

	log.Logger().Infof("Creating config for cluster %s", util.ColorInfo(c.Name()))

	path := filepath.Join(dir, Clusters, c.Name(), Terraform)
	exists, err := util.FileExists(path)
	if err != nil {
		return nil, fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
	}

	if !exists {
		log.Logger().Debugf("cluster %s does not exist, creating...", c.Name())

		os.MkdirAll(path, util.DefaultWritePermissions)

		options.Git().Clone(TerraformTemplatesGKE, path)
		g := c.(*GKECluster)
		//g := &GKECluster{}

		err := options.configureGKECluster(g, path)
		if err != nil {
			return nil, err
		}
		clusterDefinitions = append(clusterDefinitions, g)

		os.RemoveAll(filepath.Join(path, ".git"))
		os.RemoveAll(filepath.Join(path, ".gitignore"))
	} else {
		// if the directory already exists, try to load its config
		log.Logger().Debugf("cluster %s already exists, loading...", c.Name())

		g := c.(*GKECluster)
		terraformVars := filepath.Join(path, "terraform.tfvars")
		log.Logger().Infof("loading config from %s", util.ColorInfo(terraformVars))

		g.ParseTfVarsFile(terraformVars)
		clusterDefinitions = append(clusterDefinitions, g)
	}

	return clusterDefinitions, nil
}

func (options *TerraformGKEOptions) createClusters(dir string, clusterDefinitions []Cluster) error {
	log.Logger().Infof("Creating/Updating %v clusters", util.ColorInfo(len(clusterDefinitions)))
	for _, c := range clusterDefinitions {
		switch v := c.(type) {
		case *GKECluster:
			path := filepath.Join(dir, Clusters, v.Name(), Terraform)
			log.Logger().Infof("Creating/Updating cluster %s", util.ColorInfo(c.Name()))
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

func (options *TerraformGKEOptions) findDevCluster(clusterDefinitions []Cluster) (Cluster, error) {
	for _, c := range clusterDefinitions {
		if c.Name() == options.Flags.JxEnvironment {
			return c, nil
		}
	}
	return nil, fmt.Errorf("unable to find jx environment %s", options.Flags.JxEnvironment)
}

func (options *TerraformGKEOptions) writeGitIgnoreFile(dir string) error {
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		file, err := os.OpenFile(gitignore, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString("**/*.json\n.terraform\n**/*.tfstate\njx\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (options *TerraformGKEOptions) commitClusters(dir string) (bool, error) {
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

func (options *TerraformGKEOptions) configureGKECluster(g *GKECluster, path string) error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	g.DiskSize = options.Flags.GKEDiskSize
	g.AutoUpgrade = options.Flags.GKEAutoUpgrade
	g.AutoRepair = options.Flags.GKEAutoRepair
	g.MachineType = options.Flags.GKEMachineType
	g.Preemptible = options.Flags.GKEPreemptible
	g.Zone = options.Flags.GKEZone
	g.ProjectID = options.Flags.GKEProjectID
	g.MinNumOfNodes = options.Flags.GKEMinNumOfNodes
	g.MaxNumOfNodes = options.Flags.GKEMaxNumOfNodes
	g.ServiceAccount = options.Flags.GKEServiceAccount
	g.Organisation = options.Flags.OrganisationName
	g.EnableKaniko = options.InstallOptions.Flags.Kaniko
	g.EnableVault = options.InstallOptions.Flags.Vault

	if options.Flags.GKEUseEnhancedScopes {
		g.DevStorageRole = devStorageFullControl
	} else {
		g.DevStorageRole = devStorageReadOnly
	}

	if g.ServiceAccount != "" {
		log.Logger().Debugf("loading service account for cluster %s", g.Name())

		err := options.GCloud().Login(g.ServiceAccount, false)
		if err != nil {
			return err
		}
	}

	if g.ProjectID == "" {
		log.Logger().Debugf("determining google project for cluster %s", g.Name())

		projectID, err := options.getGoogleProjectID()
		if err != nil {
			return err
		}
		g.ProjectID = projectID
	}

	if !options.Flags.GKESkipEnableApis {
		log.Logger().Debugf("enabling apis for %s", g.Name())

		err := options.GCloud().EnableAPIs(g.ProjectID, "iam", "compute", "container")
		if err != nil {
			return err
		}
	}

	if g.Name() == "" {
		log.Logger().Debugf("generating a new name for cluster %s", g.Name())

		g.name = strings.ToLower(randomdata.SillyName())
		log.Logger().Infof("No cluster name provided so using a generated one: %s", util.ColorInfo(g.Name()))
	}

	if g.Zone == "" {
		log.Logger().Debugf("getting available zones for cluster %s", g.Name())

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

	if !options.BatchMode {
		if !g.Preemptible {
			prompt := &survey.Confirm{
				Message: "Would you like use preemptible VMs?",
				Default: false,
				Help:    "Preemptible VMs can significantly lower the cost of a cluster",
			}
			survey.AskOne(prompt, &g.Preemptible, nil, surveyOpts)
		}
	}

	if options.InstallOptions.Flags.NextGeneration {
		options.Flags.GKEUseEnhancedApis = true
		options.Flags.GKEUseEnhancedScopes = true
		options.InstallOptions.Flags.Kaniko = true
		options.InstallOptions.Flags.Tekton = true
		options.InstallOptions.Flags.Prow = true
		options.InstallOptions.Flags.StaticJenkins = false
	}

	if !options.BatchMode {
		if !options.Flags.GKEUseEnhancedScopes {
			prompt := &survey.Confirm{
				Message: "Would you like to access Google Cloud Storage / Google Container Registry?",
				Default: false,
				Help:    "Enables enhanced oauth scopes to allow access to storage based services",
			}
			survey.AskOne(prompt, &options.Flags.GKEUseEnhancedScopes, nil, surveyOpts)

			if options.Flags.GKEUseEnhancedScopes {
				g.DevStorageRole = devStorageFullControl
			} else {
				g.DevStorageRole = devStorageReadOnly
			}
		}
	}

	if !options.BatchMode {
		// only provide the option if enhanced scopes are enabled
		if options.Flags.GKEUseEnhancedScopes {
			if !options.Flags.GKEUseEnhancedApis {
				prompt := &survey.Confirm{
					Message: "Would you like to enable Cloud Build, Container Registry & Container Analysis APIs?",
					Default: options.Flags.GKEUseEnhancedScopes,
					Help:    "Enables extra APIs on the GCP project",
				}
				survey.AskOne(prompt, &options.Flags.GKEUseEnhancedApis, nil, surveyOpts)
			}
		}

	}

	if options.Flags.GKEUseEnhancedApis {
		err := options.GCloud().EnableAPIs(g.ProjectID, "cloudbuild", "containerregistry", "containeranalysis")
		if err != nil {
			return err
		}
	}

	if !options.BatchMode {
		// only provide the option if enhanced scopes are enabled
		if options.Flags.GKEUseEnhancedScopes && options.InstallOptions.Flags.Kaniko {
			if !options.InstallOptions.Flags.Kaniko {
				prompt := &survey.Confirm{
					Message: "Would you like to enable Kaniko for building container images",
					Default: options.Flags.GKEUseEnhancedScopes,
					Help:    "Use Kaniko for docker images",
				}
				survey.AskOne(prompt, &options.InstallOptions.Flags.Kaniko, nil, surveyOpts)
			}
		}
	}

	if options.InstallOptions.Flags.NextGeneration || options.InstallOptions.Flags.Tekton || options.InstallOptions.Flags.Kaniko {
		// lets default the docker registry to GCR
		if options.InstallOptions.Flags.DockerRegistry == "" {
			options.InstallOptions.Flags.DockerRegistry = "gcr.io"
		}

		// lets default the docker registry org to the project id
		if options.InstallOptions.Flags.DockerRegistryOrg == "" {
			options.InstallOptions.Flags.DockerRegistryOrg = g.ProjectID
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

	storageBucket := fmt.Sprintf(gkeBucketConfiguration, terraform.MinTerraformVersion, g.ProjectID, options.Flags.OrganisationName, g.Name())
	log.Logger().Debugf("Using bucket configuration %s", storageBucket)

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

		log.Logger().Infof("Created %s", terraformTf)
	}

	return nil
}

func (options *TerraformGKEOptions) applyTerraformGKE(g *GKECluster, path string) error {
	surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
	if g.ProjectID == "" {
		return errors.New("Unable to apply terraform, projectId has not been set")
	}

	log.Logger().Info("Applying Terraform changes")

	terraformVars := filepath.Join(path, "terraform.tfvars")

	if g.ServiceAccount == "" {
		if options.Flags.GKEServiceAccount != "" {
			g.ServiceAccount = options.Flags.GKEServiceAccount
			err := options.GCloud().Login(g.ServiceAccount, false)
			if err != nil {
				return err
			}

			log.Logger().Debugf("attempting to enable apis")
			err = options.GCloud().EnableAPIs(g.ProjectID, "iam", "compute", "container")
			if err != nil {
				return err
			}
		}
	}

	var serviceAccountPath string
	if g.ServiceAccount == "" {
		serviceAccountName := fmt.Sprintf("%s-%s-tf", options.Flags.OrganisationName, g.Name())
		log.Logger().Infof("No GCP service account provided, creating %s", util.ColorInfo(serviceAccountName))

		_, err := options.GCloud().GetOrCreateServiceAccount(serviceAccountName, g.ProjectID, filepath.Dir(path), gke.RequiredServiceAccountRoles)
		if err != nil {
			return err
		}
		serviceAccountPath = filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.key.json", serviceAccountName))
		log.Logger().Infof("Created GCP service account: %s", util.ColorInfo(serviceAccountPath))
	} else {
		serviceAccountPath = g.ServiceAccount
		log.Logger().Infof("Using provided GCP service account: %s", util.ColorInfo(serviceAccountPath))
	}

	// create the bucket
	bucketName := fmt.Sprintf("%s-%s-terraform-state", g.ProjectID, options.Flags.OrganisationName)
	exists, err := options.GCloud().BucketExists(g.ProjectID, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = options.GCloud().CreateBucket(g.ProjectID, bucketName, g.Region())
		if err != nil {
			return err
		}
		log.Logger().Infof("Created GCS bucket: %s in region %s", util.ColorInfo(bucketName), util.ColorInfo(g.Region()))
	}

	err = terraform.Init(path, serviceAccountPath)
	if err != nil {
		return err
	}

	plan, err := terraform.Plan(path, terraformVars, serviceAccountPath)
	if err != nil {
		return err
	}

	log.Logger().Infof(plan)

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
			log.Logger().Infof("%s", util.ColorError("It looks like this plan is destructive, aborting."))
			log.Logger().Infof("Use --ignore-terraform-warnings to override")
			return errors.New("aborting destructive plan")
		}
	}

	if !options.Flags.SkipTerraformApply {
		log.Logger().Info("Applying plan...")

		err = terraform.Apply(path, terraformVars, serviceAccountPath, options.Out, options.Err)
		if err != nil {
			return err
		}

		output, err := options.GetCommandOutput("", "gcloud", "container", "clusters", "get-credentials", g.ClusterName(), "--zone", g.Zone, "--project", g.ProjectID)
		if err != nil {
			return err
		}
		log.Logger().Info(output)
	} else {
		log.Logger().Infof("Skipping Terraform apply")
	}

	options.InstallOptions.SetInstallValues(map[string]string{
		kube.Zone:        g.Zone,
		kube.Region:      g.Region(),
		kube.ProjectID:   g.ProjectID,
		kube.ClusterName: g.ClusterName(),
	})

	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (options *TerraformGKEOptions) getGoogleProjectID() (string, error) {
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
		log.Logger().Infof("Using the only Google Cloud Project %s to create the cluster", util.ColorInfo(projectID))
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

func (options *TerraformGKEOptions) installJx(c Cluster, clusters []Cluster) error {
	log.Logger().Infof("Installing jx on cluster %s with context %s", util.ColorInfo(c.Name()), util.ColorInfo(c.Context()))

	err := options.RunCommand("kubectl", "config", "use-context", c.Context())
	if err != nil {
		return err
	}

	// check if jx is already installed
	_, err = options.FindEnvironmentNamespace(c.Name())
	if err != nil {
		// jx is missing, install,
		options.InstallOptions.Flags.DefaultEnvironmentPrefix = c.ClusterName()
		err = options.initAndInstall(c.Provider())
		if err != nil {
			return err
		}

		context, err := options.GetCommandOutput("", "kubectl", "config", "current-context")
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
	log.Logger().Info("Skipping installing jx as it appears to be already installed")

	return nil
}

func (options *TerraformGKEOptions) initAndInstall(provider string) error {
	// call jx init
	options.InstallOptions.BatchMode = options.BatchMode
	options.InstallOptions.Flags.Provider = provider

	// call jx install
	installOpts := &options.InstallOptions
	err := installOpts.Run()
	if err != nil {
		return err
	}

	return nil
}

func (options *TerraformGKEOptions) configureEnvironments(clusters []Cluster) error {

	for index, cluster := range clusters {
		if cluster.Name() != options.Flags.JxEnvironment {
			jxClient, _, err := options.JXClient()
			if err != nil {
				return err
			}
			log.Logger().Infof("Checking for environments %s on cluster %s", cluster.Name(), cluster.ClusterName())
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

func booleanAsInt(input bool) string {
	if input {
		return "1"
	}
	return "0"
}
