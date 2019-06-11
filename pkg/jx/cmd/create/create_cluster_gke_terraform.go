package create

import (
	"strings"

	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	survey "gopkg.in/AlecAivazis/survey.v1"
	git "gopkg.in/src-d/go-git.v4"

	"fmt"

	"errors"

	osUser "os/user"

	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// CreateClusterOptions the flags for running create cluster
type CreateClusterGKETerraformOptions struct {
	CreateClusterOptions

	Flags CreateClusterGKETerraformFlags
}

type CreateClusterGKETerraformFlags struct {
	AutoUpgrade bool
	ClusterName string
	//ClusterIpv4Cidr string
	//ClusterVersion  string
	DiskSize      string
	MachineType   string
	MinNumOfNodes string
	MaxNumOfNodes string
	ProjectId     string
	SkipLogin     bool
	Zone          string
	Labels        string
}

var (
	createClusterGKETerraformLong = templates.LongDesc(`
		This command creates a new Kubernetes cluster on GKE, installing required local dependencies and provisions the
		Jenkins X platform

		You can see a demo of this command here: [https://jenkins-x.io/demos/create_cluster_gke/](https://jenkins-x.io/demos/create_cluster_gke/)

		Google Kubernetes Engine is a managed environment for deploying containerized applications. It brings our latest
		innovations in developer productivity, resource efficiency, automated operations, and open source flexibility to
		accelerate your time to market.

		Google has been running production workloads in containers for over 15 years, and we build the best of what we
		learn into Kubernetes, the industry-leading open source container orchestrator which powers Kubernetes Engine.

`)

	createClusterGKETerraformExample = templates.Examples(`

		jx create cluster gke terraform

`)
)

// NewCmdCreateClusterGKETerraform creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdCreateClusterGKETerraform(commonOpts *opts.CommonOptions) *cobra.Command {
	options := CreateClusterGKETerraformOptions{
		CreateClusterOptions: createCreateClusterOptions(commonOpts, cloud.GKE),
	}
	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Create a new Kubernetes cluster on GKE using Terraform: Runs on Google Cloud",
		Long:    createClusterGKETerraformLong,
		Example: createClusterGKETerraformExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.addAuthFlags(cmd)
	options.addCreateClusterFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "", "The name of this cluster, default is a random generated name")
	//cmd.Flags().StringVarP(&options.Flags.ClusterIpv4Cidr, "cluster-ipv4-cidr", "", "", "The IP address range for the pods in this cluster in CIDR notation (e.g. 10.0.0.0/14)")
	//cmd.Flags().StringVarP(&options.Flags.ClusterVersion, optionKubernetesVersion, "v", "", "The Kubernetes version to use for the master and nodes. Defaults to server-specified")
	cmd.Flags().StringVarP(&options.Flags.DiskSize, "disk-size", "d", "100", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.AutoUpgrade, "enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.MachineType, "machine-type", "m", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.MinNumOfNodes, "min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.MaxNumOfNodes, "max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.ProjectId, "project-id", "p", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.Zone, "zone", "z", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().StringVarP(&options.Flags.Labels, "labels", "", "", "The labels to add to the cluster being created such as 'foo=bar,whatnot=123'. Label names must begin with a lowercase character ([a-z]), end with a lowercase alphanumeric ([a-z0-9]) with dashes (-), and lowercase alphanumeric ([a-z0-9]) between.")
	return cmd
}

func (o *CreateClusterGKETerraformOptions) addAuthFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	cmd.Flags().StringVarP(&o.ServiceAccount, "service-account", "", "", "Use a service account to login to GCE")
}

func (o *CreateClusterGKETerraformOptions) Run() error {
	err := o.InstallRequirements(cloud.GKE, "terraform", o.InstallOptions.InitOptions.HelmBinary())
	if err != nil {
		return err
	}

	err = o.createClusterGKETerraform()
	if err != nil {
		log.Logger().Errorf("error creating cluster %v", err)
		return err
	}

	return nil
}

func (o *CreateClusterGKETerraformOptions) createClusterGKETerraform() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	if !o.BatchMode {
		confirm := false
		prompt := &survey.Confirm{
			Message: "Creating a GKE cluster with Terraform is an experimental feature in jx.  Would you like to continue?",
		}
		survey.AskOne(prompt, &confirm, nil, surveyOpts)

		if !confirm {
			// exit at this point
			return nil
		}
	}

	err := gke.Login(o.ServiceAccount, o.Flags.SkipLogin)
	if err != nil {
		return err
	}

	projectId := o.Flags.ProjectId
	if projectId == "" {
		projectId, err = o.getGoogleProjectId()
		if err != nil {
			return err
		}
	}

	err = o.RunCommand("gcloud", "config", "set", "project", projectId)
	if err != nil {
		return err
	}

	if o.Flags.ClusterName == "" {
		clusterName := strings.ToLower(randomdata.SillyName())
		prompt := &survey.Input{
			Message: "What cluster name would you like to use",
			Default: clusterName,
		}

		err = survey.AskOne(prompt, &o.Flags.ClusterName, nil, surveyOpts)
		if err != nil {
			return err
		}
	}

	zone := o.Flags.Zone
	if zone == "" {
		availableZones, err := gke.GetGoogleZones(projectId)
		if err != nil {
			return err
		}
		prompts := &survey.Select{
			Message:  "Google Cloud Zone:",
			Options:  availableZones,
			PageSize: 10,
			Help:     "The compute zone (e.g. us-central1-a) for the cluster",
		}

		err = survey.AskOne(prompts, &zone, nil, surveyOpts)
		if err != nil {
			return err
		}
	}

	machineType := o.Flags.MachineType
	if machineType == "" {
		prompts := &survey.Select{
			Message:  "Google Cloud Machine Type:",
			Options:  gke.GetGoogleMachineTypes(),
			Help:     "We recommend a minimum of n1-standard-2 for Jenkins X,  a table of machine descriptions can be found here https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture",
			PageSize: 10,
			Default:  "n1-standard-2",
		}

		err := survey.AskOne(prompts, &machineType, nil, surveyOpts)
		if err != nil {
			return err
		}
	}

	minNumOfNodes := o.Flags.MinNumOfNodes
	if minNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Minimum number of Nodes",
			Default: "3",
			Help:    "We recommend a minimum of 3 for Jenkins X,  the minimum number of nodes to be created in each of the cluster's zones",
		}

		survey.AskOne(prompt, &minNumOfNodes, nil, surveyOpts)
	}

	maxNumOfNodes := o.Flags.MaxNumOfNodes
	if maxNumOfNodes == "" {
		prompt := &survey.Input{
			Message: "Maximum number of Nodes",
			Default: "5",
			Help:    "We recommend at least 5 for Jenkins X,  the maximum number of nodes to be created in each of the cluster's zones",
		}

		survey.AskOne(prompt, &maxNumOfNodes, nil, surveyOpts)
	}

	jxHome, err := util.ConfigDir()
	if err != nil {
		return err
	}

	clustersHome := filepath.Join(jxHome, "clusters")
	clusterHome := filepath.Join(clustersHome, o.Flags.ClusterName)
	os.MkdirAll(clusterHome, os.ModePerm)

	var keyPath string

	if o.ServiceAccount == "" {
		// check to see if a service account exists
		serviceAccount := fmt.Sprintf("jx-%s", o.Flags.ClusterName)
		log.Logger().Infof("Checking for service account %s", serviceAccount)

		keyPath, err = gke.GetOrCreateServiceAccount(serviceAccount, projectId, clusterHome, gke.RequiredServiceAccountRoles)
		if err != nil {
			return err
		}

		keyPath = filepath.Join(clusterHome, fmt.Sprintf("%s.key.json", serviceAccount))
		err = o.RunCommand("gcloud", "auth", "activate-service-account", "--key-file", keyPath)
		if err != nil {
			return err
		}
	} else {
		keyPath = o.ServiceAccount
	}

	terraformDir := filepath.Join(clusterHome, "terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		os.MkdirAll(terraformDir, os.ModePerm)
		_, err = git.PlainClone(terraformDir, false, &git.CloneOptions{
			URL:           "https://github.com/jenkins-x/terraform-jx-templates-gke",
			ReferenceName: "refs/heads/master",
			SingleBranch:  true,
			Progress:      o.Out,
		})
	}

	user, err := osUser.Current()
	if err != nil {
		return err
	}
	username := sanitizeLabel(user.Username)

	// create .tfvars file in .jx folder
	terraformVars := filepath.Join(terraformDir, "terraform.tfvars")
	o.writeKeyValueIfNotExists(terraformVars, "created_by", username)
	o.writeKeyValueIfNotExists(terraformVars, "created_timestamp", time.Now().Format("20060102150405"))
	o.writeKeyValueIfNotExists(terraformVars, "credentials", keyPath)
	o.writeKeyValueIfNotExists(terraformVars, "cluster_name", o.Flags.ClusterName)
	o.writeKeyValueIfNotExists(terraformVars, "gcp_zone", zone)
	o.writeKeyValueIfNotExists(terraformVars, "gcp_project", projectId)
	o.writeKeyValueIfNotExists(terraformVars, "min_node_count", minNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "max_node_count", maxNumOfNodes)
	o.writeKeyValueIfNotExists(terraformVars, "node_machine_type", machineType)
	o.writeKeyValueIfNotExists(terraformVars, "node_preemptible", "false")
	o.writeKeyValueIfNotExists(terraformVars, "node_disk_size", o.Flags.DiskSize)
	o.writeKeyValueIfNotExists(terraformVars, "auto_repair", "false")
	o.writeKeyValueIfNotExists(terraformVars, "auto_upgrade", strconv.FormatBool(o.Flags.AutoUpgrade))
	o.writeKeyValueIfNotExists(terraformVars, "enable_kubernetes_alpha", "false")
	o.writeKeyValueIfNotExists(terraformVars, "enable_legacy_abac", "true")
	o.writeKeyValueIfNotExists(terraformVars, "logging_service", "logging.googleapis.com")
	o.writeKeyValueIfNotExists(terraformVars, "monitoring_service", "monitoring.googleapis.com")

	args := []string{"init", terraformDir}
	err = o.RunCommand("terraform", args...)
	if err != nil {
		return err
	}

	terraformState := filepath.Join(terraformDir, "terraform.tfstate")

	args = []string{"plan",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		terraformDir}

	output, err := o.GetCommandOutput("", "terraform", args...)
	if err != nil {
		return err
	}

	log.Logger().Info("Applying plan...")

	args = []string{"apply",
		"-auto-approve",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		terraformDir}

	err = o.RunCommandVerbose("terraform", args...)
	if err != nil {
		return err
	}

	// should we setup the labels at this point?
	//gcloud container clusters update ninjacandy --update-labels ''
	args = []string{"container",
		"clusters",
		"update",
		o.Flags.ClusterName}

	labels := o.Flags.Labels
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

	err = o.RunCommand("gcloud", args...)
	if err != nil {
		return err
	}

	output, err = o.GetCommandOutput("", "gcloud", "container", "clusters", "get-credentials", o.Flags.ClusterName, "--zone", zone, "--project", projectId)
	if err != nil {
		return err
	}
	log.Logger().Info(output)

	log.Logger().Info("Initialising cluster ...")
	if o.InstallOptions.Flags.DefaultEnvironmentPrefix == "" {
		o.InstallOptions.Flags.DefaultEnvironmentPrefix = o.Flags.ClusterName
	}
	err = o.initAndInstall(cloud.GKE)
	if err != nil {
		return err
	}

	context, err := o.GetCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	log.Logger().Info(context)

	ns := o.InstallOptions.Flags.Namespace
	if ns == "" {
		_, ns, _ = o.KubeClientAndNamespace()
		if err != nil {
			return err
		}
	}

	err = o.RunCommand("kubectl", "config", "set-context", context, "--namespace", ns)
	if err != nil {
		return err
	}

	err = o.RunCommand("kubectl", "get", "ingress")
	if err != nil {
		return err
	}
	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (o *CreateClusterGKETerraformOptions) getGoogleProjectId() (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
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
		projectId = existingProjects[0]
		log.Logger().Infof("Using the only Google Cloud Project %s to create the cluster", util.ColorInfo(projectId))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		err := survey.AskOne(prompts, &projectId, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}

	if projectId == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}

func (o *CreateClusterGKETerraformOptions) writeKeyValueIfNotExists(path string, key string, value string) error {
	// file exists
	if _, err := os.Stat(path); err == nil {
		buffer, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(buffer)

		log.Logger().Debugf("Checking if %s contains %s", path, key)

		if strings.Contains(contents, key) {
			log.Logger().Debugf("Skipping %s", key)
			return nil
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	line := fmt.Sprintf("%s = \"%s\"", key, value)
	log.Logger().Debugf("Writing '%s' to %s", line, path)

	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}
