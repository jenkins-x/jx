package cmd

import (
	"io"

	"strings"

	"fmt"

	"errors"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jx/cmd/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// CreateClusterOptions the flags for running crest cluster
type CreateClusterGKEOptions struct {
	CreateClusterOptions

	Flags            CreateClusterGKEFlags
	HelmValuesConfig config.HelmValuesConfig
}

type CreateClusterGKEFlags struct {
	AutoUpgrade     bool
	ClusterName     string
	ClusterIpv4Cidr string
	ClusterVersion  string
	DiskSize        string
	ImageType       string
	MachineType     string
	NumOfNodes      string
	ProjectId       string
	SkipLogin       bool
	Zone            string
	Namespace       string
}

const CLUSTER_LIST_HEADER = "PROJECT_ID"

var (
	createClusterGKELong = templates.LongDesc(`
		This command creates a new kubernetes cluster on GKE, installing required local dependencies and provisions the
		Jenkins X platform

		Google Kubernetes Engine is a managed environment for deploying containerized applications. It brings our latest
		innovations in developer productivity, resource efficiency, automated operations, and open source flexibility to
		accelerate your time to market.

		Google has been running production workloads in containers for over 15 years, and we build the best of what we
		learn into Kubernetes, the industry-leading open source container orchestrator which powers Kubernetes Engine.

`)

	createClusterGKEExample = templates.Examples(`

		jx create cluster gke

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdCreateClusterGKE(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterGKEOptions{
		HelmValuesConfig: config.HelmValuesConfig{
			ExposeController: &config.ExposeController{},
		},
		CreateClusterOptions: CreateClusterOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
			Provider:       GKE,
			InstallOptions: createInstallOptions(f, out, errOut),
		},
	}
	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Create a new kubernetes cluster on GKE: Runs on Google Cloud",
		Long:    createClusterGKELong,
		Example: createClusterGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	config.AddExposeControllerValues(cmd, &options.HelmValuesConfig)

	cmd.Flags().StringVarP(&options.Flags.ClusterName, "cluster-name", "n", "", "The name of this cluster, default is a random generated name")
	cmd.Flags().StringVarP(&options.Flags.ClusterIpv4Cidr, "cluster-ipv4-cidr", "", "", "The IP address range for the pods in this cluster in CIDR notation (e.g. 10.0.0.0/14)")
	cmd.Flags().StringVarP(&options.Flags.ClusterVersion, "cluster-version", "v", "1.7.12-gke.1", "The Kubernetes version to use for the master and nodes. Defaults to server-specified")
	cmd.Flags().StringVarP(&options.Flags.DiskSize, "disk-size", "d", "", "Size in GB for node VM boot disks. Defaults to 100GB")
	cmd.Flags().BoolVarP(&options.Flags.AutoUpgrade, "enable-autoupgrade", "", false, "Sets autoupgrade feature for a cluster's default node-pool(s)")
	cmd.Flags().StringVarP(&options.Flags.MachineType, "machine-type", "m", "", "The type of machine to use for nodes")
	cmd.Flags().StringVarP(&options.Flags.NumOfNodes, "num-nodes", "", "", "The number of nodes to be created in each of the cluster's zones")
	cmd.Flags().StringVarP(&options.Flags.ProjectId, "project-id", "p", "", "Google Project ID to create cluster in")
	cmd.Flags().StringVarP(&options.Flags.Zone, "zone", "z", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gloud auth")
	return cmd
}

func (o *CreateClusterGKEOptions) Run() error {

	var deps []string
	d := binaryShouldBeInstalled("gcloud")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("error creating cluster on GKE, %v", err)
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

	var err error
	if !o.Flags.SkipLogin {
		err := o.runCommand("gcloud", "auth", "login", "--brief")
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

	err = o.runCommand("gcloud", "config", "set", "project", projectId)
	if err != nil {
		return err
	}

	clusterName := o.Flags.ClusterName
	if clusterName == "" {
		clusterName = strings.ToLower(randomdata.SillyName())
		log.Infof("No cluster name provided so using a generated one: %s", clusterName)
	}

	zone := o.Flags.Zone
	if zone == "" {
		prompts := &survey.Select{
			Message:  "Google Cloud Zone:",
			Options:  gke.GetGoogleZones(),
			PageSize: 10,
			Help:     "The compute zone (e.g. us-central1-a) for the cluster",
		}

		err := survey.AskOne(prompts, &zone, nil)
		if err != nil {
			return err
		}
	}

	machineType := o.Flags.MachineType
	if machineType == "" {
		prompts := &survey.Select{
			Message:  "Google Cloud Machine Type:",
			Options:  gke.GetGoogleMachineTypes(),
			Help:     "A table of machine descriptions can be found here https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture",
			PageSize: 10,
			Default:  "n1-standard-2",
		}

		err := survey.AskOne(prompts, &machineType, nil)
		if err != nil {
			return err
		}
	}

	numOfNodes := o.Flags.NumOfNodes
	if numOfNodes == "" {
		prompt := &survey.Input{
			Message: "Number of Nodes",
			Default: "2",
			Help:    "The number of nodes to be created in each of the cluster's zones",
		}

		survey.AskOne(prompt, &numOfNodes, nil)
	}

	// mandatory flags are machine type, num-nodes, zone,
	args := []string{"container", "clusters", "create", clusterName, "--zone", zone, "--num-nodes", numOfNodes, "--machine-type", machineType}

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
		args = append(args, "--enable-autoupgrade", "true")
	}

	if o.Flags.ImageType != "" {
		args = append(args, "--image-type", o.Flags.ImageType)
	}

	err = o.runCommand("gcloud", args...)
	if err != nil {
		return err
	}

	// call jx init
	initOpts := &InitOptions{
		CommonOptions: o.CommonOptions,
		Flags: InitFlags{
			Provider: GKE,
		},
	}
	err = initOpts.Run()
	if err != nil {
		return err
	}

	o.InstallOptions.Flags.Provider = GKE

	// call jx install
	installOpts := &o.InstallOptions
	err = installOpts.Run()
	if err != nil {
		return err
	}

	err = o.runCommand("gcloud", "container", "clusters", "get-credentials", clusterName, "--zone", zone, "--project", projectId)
	if err != nil {
		return err
	}

	context, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	ns := o.InstallOptions.Flags.Namespace
	if ns == "" {
		f := o.Factory
		_, ns, _ = f.CreateClient()
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

// asks to chose from existing projects or optionally creates one if none exist
func (o *CreateClusterGKEOptions) getGoogleProjectId() (string, error) {
	out, err := o.getCommandOutput("", "gcloud", "projects", "list")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	var existingProjects []string
	for _, l := range lines {
		if strings.Contains(l, CLUSTER_LIST_HEADER) {
			continue
		}
		fields := strings.Fields(l)
		existingProjects = append(existingProjects, fields[0])
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
		return "", errors.New("no google project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}
