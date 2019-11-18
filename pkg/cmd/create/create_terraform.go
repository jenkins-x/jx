package create

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/features"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"fmt"

	"strconv"

	"errors"

	"path/filepath"

	"os"

	"path"

	randomdata "github.com/Pallinder/go-randomdata"
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
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// Flags for a cluster
type Flags struct {
	Cluster                     []string
	ClusterName                 string // cannot be used in conjunction with Cluster
	CloudProvider               string // cannot be used in conjunction with Cluster
	OrganisationName            string
	SkipLogin                   bool
	ForkOrganisationGitRepo     string
	SkipTerraformApply          bool
	SkipInstallation            bool
	NoActiveCluster             bool
	IgnoreTerraformWarnings     bool
	JxEnvironment               string
	GKEProjectID                string
	GKESkipEnableApis           bool
	GKEZone                     string
	GKEMachineType              string
	GKEPreemptible              bool
	GKEMinNumOfNodes            string
	GKEMaxNumOfNodes            string
	GKEDiskSize                 string
	GKEAutoRepair               bool
	GKEAutoUpgrade              bool
	GKEServiceAccount           string
	GKEUseEnhancedScopes        bool
	GKEUseEnhancedApis          bool
	LocalOrganisationRepository string
}

// CreateTerraformOptions the options for the create spring command
type CreateTerraformOptions struct {
	CreateOptions
	InstallOptions InstallOptions
	Flags          Flags
	Clusters       []Cluster
}

var (
	createTerraformExample = templates.Examples(`
		jx create terraform

		# to specify the clusters via flags
		jx create terraform -c dev=gke -c stage=gke -c prod=gke

`)
)

// NewCmdCreateTerraform creates a command object for the "create" command
func NewCmdCreateTerraform(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateTerraformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
		InstallOptions: CreateInstallOptions(commonOpts),
	}

	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Creates a Jenkins X Terraform plan",
		Example: createTerraformExample,
		PreRun: func(cmd *cobra.Command, args []string) {
			err := features.IsEnabled(cmd)
			helper.CheckErr(err)
			err = options.InstallOptions.CheckFeatures()
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

	// add sub commands
	cmd.AddCommand(NewCmdCreateTerraformGKE(commonOpts))

	return cmd
}

func (options *CreateTerraformOptions) addFlags(cmd *cobra.Command, addSharedFlags bool) {
	// global flags
	if addSharedFlags {
		cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	}
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, optionCluster, "c", []string{}, "Name and Kubernetes provider (currently gke only) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "", "", "The name of a single cluster to create - cannot be used in conjunction with --"+optionCluster)
	cmd.Flags().StringVarP(&options.Flags.CloudProvider, optionCloudProvider, "", "", "The cloud provider (currently gke only) - cannot be used in conjunction with --"+optionCluster)
	cmd.Flags().BoolVarP(&options.Flags.SkipTerraformApply, "skip-terraform-apply", "", false, "Skip applying the generated Terraform plans")
	cmd.Flags().BoolVarP(&options.Flags.SkipInstallation, "skip-installation", "", false, "Provision cluster(s) only, don't install Jenkins X into it")
	cmd.Flags().BoolVarP(&options.Flags.IgnoreTerraformWarnings, "ignore-terraform-warnings", "", false, "Ignore any warnings about the Terraform plan being potentially destructive")
	cmd.Flags().StringVarP(&options.Flags.JxEnvironment, "jx-environment", "", "dev", "The cluster name to install jx inside")
	cmd.Flags().StringVarP(&options.Flags.LocalOrganisationRepository, "local-organisation-repository", "", "", "Rather than cloning from a remote Git server, the local directory to use for the organisational folder")
	cmd.Flags().BoolVarP(&options.Flags.NoActiveCluster, "no-active-cluster", "", false, "Tells JX there's isn't currently an active cluster, so we cannot use it for configuration")

	// GKE specific overrides
	cmd.Flags().StringVarP(&options.Flags.GKEDiskSize, "gke-disk-size", "", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoUpgrade, "gke-enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoRepair, "gke-enable-autorepair", "", true, "Sets autorepair feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEPreemptible, "gke-preemptible", "", false, "Use preemptible VMs in the node-pool")
	cmd.Flags().StringVarP(&options.Flags.GKEMachineType, "gke-machine-type", "", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.GKEMinNumOfNodes, "gke-min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEMaxNumOfNodes, "gke-max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEProjectID, "gke-project-id", "", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.GKEZone, "gke-zone", "", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().BoolVarP(&options.Flags.GKEUseEnhancedScopes, "gke-use-enhanced-scopes", "", false, "Use enhanced Oauth scopes for access to GCS/GCR")
	cmd.Flags().BoolVarP(&options.Flags.GKEUseEnhancedApis, "gke-use-enhanced-apis", "", false, "Enable enhanced APIs to utilise Container Registry & Cloud Build")
}

// Run implements this command
func (options *CreateTerraformOptions) Run() error {
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

	err = terraform.CheckVersion()
	if err != nil {
		return err
	}

	err = options.InstallOptions.InitOptions.ValidateGit()
	if err != nil {
		return err
	}

	err = options.ValidateClusterDetails()
	if err != nil {
		return err
	}

	if len(options.Clusters) == 0 {
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
	if len(options.Flags.Cluster) > 0 {
		if options.Flags.ClusterName != "" || options.Flags.CloudProvider != "" {
			return fmt.Errorf("--%s cannot be used in conjunction with --%s or --%s", optionCluster, optionClusterName, optionCloudProvider)
		}
		for _, p := range options.Flags.Cluster {
			pair := strings.Split(p, "=")
			if len(pair) != 2 {
				return fmt.Errorf("need to provide cluster values as --%s name=provider, e.g. --%s production=gke", optionCluster, optionCluster)
			}
			if !stringInValidProviders(pair[1]) {
				return fmt.Errorf("invalid cluster provider type %s, must be one of %v", p, validTerraformClusterProviders)
			}

			c := &GKECluster{name: pair[0], provider: pair[1]}
			options.Clusters = append(options.Clusters, c)
		}
	} else if options.Flags.ClusterName != "" || options.Flags.CloudProvider != "" {
		options.Clusters = []Cluster{&GKECluster{name: options.Flags.ClusterName, provider: options.Flags.CloudProvider}}
	}
	return nil
}

func (options *CreateTerraformOptions) createOrganisationGitRepo() error {
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
func (options *CreateTerraformOptions) CreateOrganisationFolderStructure(dir string) ([]Cluster, error) {
	options.writeGitIgnoreFile(dir)

	clusterDefinitions := []Cluster{}

	for _, c := range options.Clusters {
		log.Logger().Infof("Creating config for cluster %s\n", util.ColorInfo(c.Name()))

		path := filepath.Join(dir, Clusters, c.Name(), Terraform)
		exists, err := util.FileExists(path)
		if err != nil {
			return nil, fmt.Errorf("unable to check if existing folder exists for path %s: %v", path, err)
		}

		if !exists {
			log.Logger().Debugf("cluster %s does not exist, creating...", c.Name())

			os.MkdirAll(path, util.DefaultWritePermissions)

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
				return nil, fmt.Errorf("unknown Kubernetes provider type '%s' must be one of %v", c.Provider(), validTerraformClusterProviders)
			}
			os.RemoveAll(filepath.Join(path, ".git"))
			os.RemoveAll(filepath.Join(path, ".gitignore"))
		} else {
			// if the directory already exists, try to load its config
			log.Logger().Debugf("cluster %s already exists, loading...", c.Name())

			switch c.Provider() {
			case "gke", "jx-infra":
				//g := &GKECluster{}
				g := c.(*GKECluster)
				terraformVars := filepath.Join(path, "terraform.tfvars")
				log.Logger().Infof("loading config from %s", util.ColorInfo(terraformVars))

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

func (options *CreateTerraformOptions) findDevCluster(clusterDefinitions []Cluster) (Cluster, error) {
	for _, c := range clusterDefinitions {
		if c.Name() == options.Flags.JxEnvironment {
			return c, nil
		}
	}
	return nil, fmt.Errorf("unable to find jx environment %s", options.Flags.JxEnvironment)
}

func (options *CreateTerraformOptions) writeGitIgnoreFile(dir string) error {
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

func (options *CreateTerraformOptions) applyTerraformGKE(g *GKECluster, path string) error {
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

func (options *CreateTerraformOptions) installJx(c Cluster, clusters []Cluster) error {
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

func (options *CreateTerraformOptions) initAndInstall(provider string) error {
	// call jx init
	options.InstallOptions.BatchMode = options.BatchMode
	options.InstallOptions.Flags.Provider = provider

	if len(options.Clusters) > 1 {
		options.InstallOptions.Flags.NoDefaultEnvironments = true
		log.Logger().Info("Creating custom environments in each cluster")
	} else {
		log.Logger().Info("Creating default environments")
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
