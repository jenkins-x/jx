package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/pkg/errors"

	osUser "os/user"

	"regexp"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateClusterOptions the flags for running create cluster
type CreateClusterGKEOptions struct {
	CreateClusterOptions

	Flags CreateClusterGKEFlags
}

type CreateClusterGKEFlags struct {
	AutoUpgrade     bool
	ClusterName     string
	ClusterIpv4Cidr string
	ClusterVersion  string
	DiskSize        string
	ImageType       string
	MachineType     string
	MinNumOfNodes   string
	MaxNumOfNodes   string
	Network         string
	ProjectId       string
	SkipLogin       bool
	SubNetwork      string
	Zone            string
	Namespace       string
	Labels          string
	Scopes          []string
	Preemptible     bool
}

const clusterListHeader = "PROJECT_ID"

var (
	createClusterGKELong = templates.LongDesc(`
		This command creates a new Kubernetes cluster on GKE, installing required local dependencies and provisions the
		Jenkins X platform

		You can see a demo of this command here: [https://jenkins-x.io/demos/create_cluster_gke/](https://jenkins-x.io/demos/create_cluster_gke/)

		Google Kubernetes Engine is a managed environment for deploying containerized applications. It brings our latest
		innovations in developer productivity, resource efficiency, automated operations, and open source flexibility to
		accelerate your time to market.

		Google has been running production workloads in containers for over 15 years, and we build the best of what we
		learn into Kubernetes, the industry-leading open source container orchestrator which powers Kubernetes Engine.

`)

	createClusterGKEExample = templates.Examples(`

		jx create cluster gke

`)
	disallowedLabelCharacters = regexp.MustCompile("[^a-z0-9-]")
)

// NewCmdCreateClusterGKE creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdCreateClusterGKE(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := CreateClusterGKEOptions{
		CreateClusterOptions: createCreateClusterOptions(f, in, out, errOut, GKE),
	}
	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Create a new Kubernetes cluster on GKE: Runs on Google Cloud",
		Long:    createClusterGKELong,
		Example: createClusterGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "", "The name of this cluster, default is a random generated name")
	cmd.Flags().StringVarP(&options.Flags.ClusterIpv4Cidr, "cluster-ipv4-cidr", "", "", "The IP address range for the pods in this cluster in CIDR notation (e.g. 10.0.0.0/14)")
	cmd.Flags().StringVarP(&options.Flags.ClusterVersion, optionKubernetesVersion, "v", "", "The Kubernetes version to use for the master and nodes. Defaults to server-specified")
	cmd.Flags().StringVarP(&options.Flags.DiskSize, "disk-size", "d", "", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.AutoUpgrade, "enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.MachineType, "machine-type", "m", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.MinNumOfNodes, "min-num-nodes", "", "", "The minimum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.MaxNumOfNodes, "max-num-nodes", "", "", "The maximum number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.Network, "network", "", "", "The Compute Engine Network that the cluster will connect to")
	cmd.Flags().StringVarP(&options.Flags.ProjectId, "project-id", "p", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.SubNetwork, "subnetwork", "", "", "The Google Compute Engine subnetwork to which the cluster is connected")
	cmd.Flags().StringVarP(&options.Flags.Zone, "zone", "z", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	cmd.Flags().StringVarP(&options.Flags.Labels, "labels", "", "", "The labels to add to the cluster being created such as 'foo=bar,whatnot=123'. Label names must begin with a lowercase character ([a-z]), end with a lowercase alphanumeric ([a-z0-9]) with dashes (-), and lowercase alphanumeric ([a-z0-9]) between.")
	cmd.Flags().StringArrayVarP(&options.Flags.Scopes, "scope", "", []string{}, "The OAuth scopes to be added to the cluster")
	cmd.Flags().BoolVarP(&options.Flags.Preemptible, "preemptible", "", false, "Use preemptible VMs in the node-pool")

	cmd.AddCommand(NewCmdCreateClusterGKETerraform(f, in, out, errOut))

	return cmd
}

func (o *CreateClusterGKEOptions) Run() error {
	err := o.installRequirements(GKE)
	if err != nil {
		return err
	}

	err = o.createClusterGKE()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		return err
	}

	return nil
}

func (o *CreateClusterGKEOptions) createClusterGKE() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	var err error
	if !o.Flags.SkipLogin {
		err := o.runCommandVerbose("gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}

	projectId := o.Flags.ProjectId
	if projectId == "" {
		projectId, err = o.getGoogleProjectId()
		if err != nil {
			return err
		}
	}

	err = o.runCommandVerbose("gcloud", "config", "set", "project", projectId)
	if err != nil {
		return err
	}

	// lets ensure we've got container and compute enabled
	glcoudArgs := []string{"services", "enable", "container", "compute"}
	log.Infof("Let's ensure we have container and compute enabled on your project via: %s\n", util.ColorInfo("gcloud "+strings.Join(glcoudArgs, " ")))

	err = o.runCommandVerbose("gcloud", glcoudArgs...)
	if err != nil {
		return err
	}

	if o.Flags.ClusterName == "" {
		o.Flags.ClusterName = strings.ToLower(randomdata.SillyName())
		log.Infof("No cluster name provided so using a generated one: %s\n", o.Flags.ClusterName)
	}

	zone := o.Flags.Zone
	if zone == "" {
		zone, err = o.getGoogleZone(projectId)
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

	// mandatory flags are machine type, num-nodes, zone,
	args := []string{"container", "clusters", "create",
		o.Flags.ClusterName, "--zone", zone,
		"--num-nodes", minNumOfNodes,
		"--machine-type", machineType,
		"--enable-autoscaling",
		"--min-nodes", minNumOfNodes,
		"--max-nodes", maxNumOfNodes}

	if o.Flags.DiskSize != "" {
		args = append(args, "--disk-size", o.Flags.DiskSize)
	}

	if o.Flags.ClusterIpv4Cidr != "" {
		args = append(args, "--cluster-ipv4-cidr", o.Flags.ClusterIpv4Cidr)
	}

	if o.Flags.ClusterVersion != "" {
		args = append(args, "--cluster-version", o.Flags.ClusterVersion)
	}

	if o.Flags.AutoUpgrade {
		args = append(args, "--enable-autoupgrade")
	}

	if o.Flags.ImageType != "" {
		args = append(args, "--image-type", o.Flags.ImageType)
	}

	if o.Flags.Network != "" {
		args = append(args, "--network", o.Flags.Network)
	}

	if o.Flags.SubNetwork != "" {
		args = append(args, "--subnetwork", o.Flags.SubNetwork)
	}

	if len(o.Flags.Scopes) > 0 {
		args = append(args, fmt.Sprintf("--scopes=%s", strings.Join(o.Flags.Scopes, ",")))
	}

	if o.Flags.Preemptible {
		args = append(args, "--preemptible")
	}

	labels := o.Flags.Labels
	user, err := osUser.Current()
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
	if labels != "" {
		args = append(args, "--labels="+strings.ToLower(labels))
	}

	log.Info("Creating cluster...\n")
	err = o.RunCommand("gcloud", args...)
	if err != nil {
		return err
	}

	log.Info("Initialising cluster ...\n")
	if o.InstallOptions.Flags.DefaultEnvironmentPrefix == "" {
		o.InstallOptions.Flags.DefaultEnvironmentPrefix = o.Flags.ClusterName
	}
	err = o.initAndInstall(GKE)
	if err != nil {
		return err
	}

	err = o.RunCommand("gcloud", "container", "clusters", "get-credentials", o.Flags.ClusterName, "--zone", zone, "--project", projectId)
	if err != nil {
		return err
	}

	context, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	kubeClient, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	if o.InstallOptions.Flags.Namespace != "" {
		ns = o.InstallOptions.Flags.Namespace
	}

	err = o.RunCommand("kubectl", "config", "set-context", context, "--namespace", ns)
	if err != nil {
		return err
	}

	err = o.RunCommand("kubectl", "get", "ingress")
	if err != nil {
		return err
	}

	// TODO - Reenable vault for GitOpsMode
	//o.CreateClusterOptions.InstallOptions.GitOpsMode || o.CreateClusterOptions.InstallOptions.Vault {
	if o.CreateClusterOptions.InstallOptions.Vault {
		if err = InstallVaultOperator(&o.CommonOptions, ""); err != nil {
			return err
		}
		err = secrets.NewSecretLocation(kubeClient, ns).SetInVault(true)
		if err != nil {
			return errors.Wrap(err, "configring secrets location")
		}
	}

	return nil
}

func sanitizeLabel(username string) string {
	sanitized := strings.ToLower(username)
	return disallowedLabelCharacters.ReplaceAllString(sanitized, "-")
}
