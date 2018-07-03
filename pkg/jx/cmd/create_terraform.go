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
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"io/ioutil"
	"time"
)

type Cluster interface {
	Name() string
	Provider() string
}

type GKECluster struct {
	_Name         string
	_Provider     string
	ProjectId     string
	Zone          string
	MachineType   string
	MinNumOfNodes string
	MaxNumOfNodes string
	DiskSize      string
	AutoRepair    bool
	AutoUpgrade   bool
}

func (c GKECluster) Name() string {
	return c._Name
}

func (c GKECluster) Provider() string {
	return c._Provider
}

type Flags struct {
	Cluster                 []string
	OrganisationName        string
	ForkOrganisationGitRepo string
	SkipTerraformApply      bool
	JxEnvironment           string
	GKEProjectId            string
	GKEZone                 string
	GKEMachineType          string
	GKEMinNumOfNodes        string
	GKEMaxNumOfNodes        string
	GKEDiskSize             string
	GKEAutoRepair           bool
	GKEAutoUpgrade          bool
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
	addGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)

	return cmd
}

func (options *CreateTerraformOptions) addFlags(cmd *cobra.Command) {
	// global flags
	cmd.Flags().StringArrayVarP(&options.Flags.Cluster, "cluster", "c", []string{}, "Name and Kubernetes provider (gke, aks, eks) of clusters to be created in the form --cluster foo=gke")
	cmd.Flags().StringVarP(&options.Flags.OrganisationName, "organisation-name", "o", "", "The organisation name that will be used as the Git repo containing cluster details, the repo will be organisation-<org name>")
	cmd.Flags().StringVarP(&options.Flags.ForkOrganisationGitRepo, "fork-git-repo", "f", kube.DefaultOrganisationGitRepoURL, "The Git repository used as the fork when creating new Organisation git repos")
	cmd.Flags().BoolVarP(&options.Flags.SkipTerraformApply, "skip-terraform-apply", "", false, "Skip applying the generated terraform plans")
	cmd.Flags().StringVarP(&options.Flags.JxEnvironment, "jx-environment", "", "dev", "The cluster name to install jx inside")

	// gke specific overrides
	cmd.Flags().StringVarP(&options.Flags.GKEDiskSize, "gke-disk-size", "", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoUpgrade, "gke-enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().BoolVarP(&options.Flags.GKEAutoRepair, "gke-enable-autorepair", "", true, "Sets autorepair feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.GKEMachineType, "gke-machine-type", "", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.GKEMinNumOfNodes, "gke-min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEMaxNumOfNodes, "gke-max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.GKEProjectId, "gke-project-id", "", "", "Google Project ID to create cluster in")
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
func (o *CreateTerraformOptions) Run() error {

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

		c := GKECluster{_Name: name, _Provider: provider}

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
	} else {
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

	fmt.Fprintf(o.Stdout(), "Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

	if !o.Flags.SkipTerraformApply {
		fmt.Fprintf(o.Stdout(), "Applying terraform changes\n")
		err = o.createClusters(dir, clusterDefinitions)
		if err != nil {
			return err
		}

		//err = o.installJx(dir, clusterDefinitions)
		//if err != nil {
		//	return err
		//}
	} else {
		fmt.Fprintf(o.Stdout(), "Skipping terraform apply\n")
	}

	// if the cluster is called dev, install jx

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
		}
	}

	return clusterDefinitions, nil
}

func (o *CreateTerraformOptions) createClusters(dir string, clusterDefinitions []Cluster) error {
	fmt.Printf("Creating/Updating %v clusters\n", len(clusterDefinitions))
	for _, c := range clusterDefinitions {
		switch v := c.(type) {
		case GKECluster:
			path := filepath.Join(dir, Clusters, v.Name(), Terraform)
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

func (o *CreateTerraformOptions) configureGKECluster(g *GKECluster, path string) (error) {
	g.DiskSize = o.Flags.GKEDiskSize
	g.AutoUpgrade = o.Flags.GKEAutoUpgrade
	g.AutoRepair = o.Flags.GKEAutoRepair
	g.MachineType = o.Flags.GKEMachineType
	g.Zone = o.Flags.GKEZone
	g.ProjectId = o.Flags.GKEProjectId
	g.MinNumOfNodes = o.Flags.GKEMinNumOfNodes
	g.MaxNumOfNodes = o.Flags.GKEMaxNumOfNodes

	if g.ProjectId == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		g.ProjectId = projectId
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

	user, err := os_user.Current()
	var username string
	if err != nil {
		username = "unknown"
	} else {
		username = sanitizeLabel(user.Username)
	}

	terraformVars := filepath.Join(path, "terraform.tfvars")
	o.writeKeyValueIfNotExists(terraformVars, "created_by", username)
	o.writeKeyValueIfNotExists(terraformVars, "created_timestamp", time.Now().Format("20060102150405"))
	o.writeKeyValueIfNotExists(terraformVars, "cluster_name", g.Name())
	o.writeKeyValueIfNotExists(terraformVars, "gcp_zone", g.Zone)
	o.writeKeyValueIfNotExists(terraformVars, "gcp_project", g.ProjectId)
	o.writeKeyValueIfNotExists(terraformVars, "min_node_count", g.MinNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "max_node_count", g.MaxNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "node_machine_type", g.MachineType)
	o.writeKeyValueIfNotExists(terraformVars, "node_preemptible", "false")
	o.writeKeyValueIfNotExists(terraformVars, "node_disk_size", g.DiskSize)
	o.writeKeyValueIfNotExists(terraformVars, "auto_repair", strconv.FormatBool(g.AutoRepair))
	o.writeKeyValueIfNotExists(terraformVars, "auto_upgrade", strconv.FormatBool(g.AutoUpgrade))
	o.writeKeyValueIfNotExists(terraformVars, "enable_kubernetes_alpha", "false")
	o.writeKeyValueIfNotExists(terraformVars, "enable_legacy_abac", "true")
	o.writeKeyValueIfNotExists(terraformVars, "logging_service", "logging.googleapis.com")
	o.writeKeyValueIfNotExists(terraformVars, "monitoring_service", "monitoring.googleapis.com")

	return nil
}

func (o *CreateTerraformOptions) applyTerraformGKE(g *GKECluster, path string) error {
	log.Info("Applying Terraform changes\n")
	user, err := os_user.Current()
	if err != nil {
		return err
	}

	terraformVars := filepath.Join(path, "terraform.tfvars")
	serviceAccountName := fmt.Sprintf("jx-%s-%s", o.Flags.OrganisationName, g.Name())
	// create service account
	_, err = gke.GetOrCreateServiceAccount(serviceAccountName, g.ProjectId, filepath.Dir(path))
	if err != nil {
		return err
	}
	serviceAccountPath := filepath.Join(filepath.Dir(path), fmt.Sprintf("%s.key.json", serviceAccountName))

	args := []string{"init", path}
	err = o.runCommand("terraform", args...)
	if err != nil {
		return err
	}

	terraformState := filepath.Join(path, "terraform.tfstate")

	args = []string{"plan",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		"-var",
		fmt.Sprintf("credentials=%s", serviceAccountPath),
		path}

	err = o.runCommand("terraform", args...)
	if err != nil {
		return err
	}

	log.Info("Applying plan...\n")

	args = []string{"apply",
		"-auto-approve",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		"-var",
		fmt.Sprintf("credentials=%s", serviceAccountPath),
		path}

	err = o.runCommandVerbose("terraform", args...)
	if err != nil {
		return err
	}

	// should we setup the labels at this point?
	//gcloud container clusters update ninjacandy --update-labels ''
	args = []string{"container",
		"clusters",
		"update",
		g.Name()}

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

	output, err := o.getCommandOutput("", "gcloud", "container", "clusters", "get-credentials", g.Name(), "--zone", g.Zone, "--project", g.ProjectId)
	if err != nil {
		return err
	}
	log.Info(output)

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

func (o *CreateTerraformOptions) installJx(c Cluster) error {
	log.Info("Initialising cluster ...\n")
	o.InstallOptions.Flags.DefaultEnvironmentPrefix = c.Name()
	err := o.initAndInstall(c.Provider())
	if err != nil {
		return err
	}

	context, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	log.Info(context)

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
	return nil
}

func (o *CreateTerraformOptions) initAndInstall(provider string) error {
	// call jx init
	o.InstallOptions.BatchMode = o.BatchMode
	o.InstallOptions.Flags.Provider = provider

	// call jx install
	installOpts := &o.InstallOptions

	err := installOpts.Run()
	if err != nil {
		return err
	}
	return nil
}
