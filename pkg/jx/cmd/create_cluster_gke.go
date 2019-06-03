package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/Pallinder/go-randomdata"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/features"
	"github.com/jenkins-x/jx/pkg/kube"
	"gopkg.in/AlecAivazis/survey.v1"

	osUser "os/user"

	"regexp"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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
	Region          string
	Zone            string
	Namespace       string
	Labels          string
	EnhancedScopes  bool
	Scopes          []string
	Preemptible     bool
	EnhancedApis    bool
}

const (
	preemptibleFlagName    = "preemptible"
	enhancedAPIFlagName    = "enhanced-apis"
	enhancedScopesFlagName = "enhanced-scopes"
)

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
func NewCmdCreateClusterGKE(commonOpts *opts.CommonOptions) *cobra.Command {
	options := CreateClusterGKEOptions{
		CreateClusterOptions: createCreateClusterOptions(commonOpts, cloud.GKE),
	}
	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "Create a new Kubernetes cluster on GKE: Runs on Google Cloud",
		Long:    createClusterGKELong,
		Example: createClusterGKEExample,
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

	options.addCreateClusterFlags(cmd)

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
	cmd.Flags().StringVarP(&options.Flags.ImageType, "image-type", "", "", "The image type for the nodes in the cluster")
	cmd.Flags().StringVarP(&options.Flags.SubNetwork, "subnetwork", "", "", "The Google Compute Engine subnetwork to which the cluster is connected")
	cmd.Flags().StringVarP(&options.Flags.Zone, "zone", "z", "", "The compute zone (e.g. us-central1-a) for the cluster")
	cmd.Flags().StringVarP(&options.Flags.Region, "region", "r", "", "Compute region (e.g. us-central1) for the cluster")
	cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gcloud auth")
	cmd.Flags().StringVarP(&options.Flags.Labels, "labels", "", "", "The labels to add to the cluster being created such as 'foo=bar,whatnot=123'. Label names must begin with a lowercase character ([a-z]), end with a lowercase alphanumeric ([a-z0-9]) with dashes (-), and lowercase alphanumeric ([a-z0-9]) between.")
	cmd.Flags().StringArrayVarP(&options.Flags.Scopes, "scope", "", []string{}, "The OAuth scopes to be added to the cluster")
	cmd.Flags().BoolVarP(&options.Flags.Preemptible, preemptibleFlagName, "", false, "Use preemptible VMs in the node-pool")
	cmd.Flags().BoolVarP(&options.Flags.EnhancedScopes, enhancedScopesFlagName, "", false, "Use enhanced Oauth scopes for access to GCS/GCR")
	cmd.Flags().BoolVarP(&options.Flags.EnhancedApis, enhancedAPIFlagName, "", false, "Enable enhanced APIs to utilise Container Registry & Cloud Build")

	cmd.AddCommand(NewCmdCreateClusterGKETerraform(commonOpts))

	return cmd
}

func (o *CreateClusterGKEOptions) Run() error {
	// Issue 3251
	err := validateClusterName(o.Flags.ClusterName)
	if err != nil {
		return err
	}

	err = o.InstallRequirements(cloud.GKE)
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
		err := o.RunCommandVerbose("gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}

	projectId := o.Flags.ProjectId
	if projectId == "" {
		projectId, err = o.GetGoogleProjectId()
		if err != nil {
			return err
		}
	}

	err = o.RunCommandVerbose("gcloud", "config", "set", "project", projectId)
	if err != nil {
		return err
	}

	log.Infof("Let's ensure we have %s and %s enabled on your project\n", util.ColorInfo("container"), util.ColorInfo("compute"))
	err = gke.EnableAPIs(projectId, "container", "compute")
	if err != nil {
		return err
	}

	advanced := o.InstallOptions.Flags.Advanced

	clusterName := o.Flags.ClusterName
	if clusterName == "" {
		defaultClusterName := strings.ToLower(randomdata.SillyName())
		if advanced {
			prompt := &survey.Input{
				Message: "What cluster name would you like to use",
				Default: defaultClusterName,
			}

			err = survey.AskOne(prompt, &clusterName, nil, surveyOpts)
			if err != nil {
				return err
			}
		} else {
			clusterName = defaultClusterName
			log.Infof("No cluster name provided so using a generated one: %s", util.ColorPrompt(clusterName))
		}
	}

	region := o.Flags.Region
	zone := o.Flags.Zone

	if o.InstallOptions.Flags.NextGeneration && region != "" {
		return errors.New("cannot create a regional cluster with --ng")
	}

	if !o.BatchMode {
		if zone == "" && region == "" {
			clusterType := "Zonal"

			if o.InstallOptions.Flags.NextGeneration {
				log.Infof(util.ColorWarning("Defaulting to zonal cluster type as --ng is selected.\n"))
			} else if advanced {
				prompts := &survey.Select{
					Message: "What type of cluster would you like to create",
					Options: []string{"Regional", "Zonal"},
					Help:    "A Regional cluster will create a node-pool in each zone causing it to use more resources.  Please ensure you have enough quota.",
					Default: clusterType,
				}

				err = survey.AskOne(prompts, &clusterType, nil, surveyOpts)
				if err != nil {
					return err
				}
			} else {
				log.Infof("Defaulting to cluster type: %s", util.ColorPrompt(clusterType))
			}

			if "Regional" == clusterType {
				region, err = o.GetGoogleRegion(projectId)
				if err != nil {
					return err
				}
			} else {
				zone, err = o.GetGoogleZone(projectId)
				if err != nil {
					return err
				}
			}
		}
	} else {
		if zone == "" && region == "" {
			return errors.New("in batchmode, either a region or a zone must be set")
		}
	}

	machineType := o.Flags.MachineType
	if machineType == "" {
		defaultMachineType := "n1-standard-2"
		if advanced {
			prompts := &survey.Select{
				Message:  "Google Cloud Machine Type:",
				Options:  gke.GetGoogleMachineTypes(),
				Help:     "We recommend a minimum of n1-standard-2 for Jenkins X,  a table of machine descriptions can be found here https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture",
				PageSize: 10,
				Default:  defaultMachineType,
			}
			err := survey.AskOne(prompts, &machineType, nil, surveyOpts)

			if err != nil {
				return err
			}
		} else {
			machineType = defaultMachineType
			log.Infof("Defaulting to machine type: %s", util.ColorPrompt(machineType))
		}
	}

	minNumOfNodes := o.Flags.MinNumOfNodes
	if minNumOfNodes == "" {
		defaultNodes := "3"
		if region != "" {
			defaultNodes = "1"
		}
		if advanced {
			prompt := &survey.Input{
				Message: "Minimum number of Nodes (per zone)",
				Default: defaultNodes,
				Help:    "We recommend a minimum of " + defaultNodes + " for Jenkins X, the minimum number of nodes to be created in each of the cluster's zones",
			}

			err = survey.AskOne(prompt, &minNumOfNodes, nil, surveyOpts)
			if err != nil {
				return err
			}
		} else {
			minNumOfNodes = defaultNodes
			log.Infof("Defaulting to minimum number of nodes: %s", util.ColorPrompt(minNumOfNodes))
		}
	}

	maxNumOfNodes := o.Flags.MaxNumOfNodes
	if maxNumOfNodes == "" {
		defaultNodes := "5"
		if region != "" {
			defaultNodes = "2"
		}
		if advanced {
			prompt := &survey.Input{
				Message: "Maximum number of Nodes",
				Default: defaultNodes,
				Help:    "We recommend at least " + defaultNodes + " for Jenkins X, the maximum number of nodes to be created in each of the cluster's zones",
			}

			err = survey.AskOne(prompt, &maxNumOfNodes, nil, surveyOpts)
			if err != nil {
				return err
			}
		} else {
			maxNumOfNodes = defaultNodes
			log.Infof("Defaulting to maxiumum number of nodes: %s", util.ColorPrompt(maxNumOfNodes))
		}
	}

	if !o.BatchMode {
		if !o.IsFlagExplicitlySet(preemptibleFlagName) {
			if advanced {
				prompt := &survey.Confirm{
					Message: "Would you like to use preemptible VMs?",
					Default: false,
					Help:    "Preemptible VMs can significantly lower the cost of a cluster",
				}
				err = survey.AskOne(prompt, &o.Flags.Preemptible, nil, surveyOpts)
				if err != nil {
					return err
				}
			} else {
				o.Flags.Preemptible = false
				log.Infof("Defaulting use of preemptible VMs: %v", util.ColorPrompt(util.YesNo(o.Flags.Preemptible)))
			}
		}
	}

	if o.InstallOptions.Flags.NextGeneration || o.InstallOptions.Flags.Tekton {
		o.Flags.EnhancedApis = true
		o.Flags.EnhancedScopes = true
		o.InstallOptions.Flags.Kaniko = true
	}

	if !o.BatchMode {
		// if scopes is empty &
		if len(o.Flags.Scopes) == 0 && !o.IsFlagExplicitlySet(enhancedScopesFlagName) {
			if advanced {
				prompt := &survey.Confirm{
					Message: "Would you like to access Google Cloud Storage / Google Container Registry?",
					Default: o.InstallOptions.Flags.DockerRegistry == "",
					Help:    "Enables enhanced oauth scopes to allow access to storage based services",
				}
				err = survey.AskOne(prompt, &o.Flags.EnhancedScopes, nil, surveyOpts)
				if err != nil {
					return err
				}
			} else {
				o.Flags.EnhancedScopes = true
				log.Infof("Defaulting access to Google Cloud Storage / Google Container Registry: %v", util.ColorPrompt(util.YesNo(o.Flags.EnhancedScopes)))
			}
		}
	}

	if o.Flags.EnhancedScopes {
		o.Flags.Scopes = []string{"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/devstorage.full_control",
			"https://www.googleapis.com/auth/service.management",
			"https://www.googleapis.com/auth/servicecontrol",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring"}
	}

	if !o.BatchMode {
		// only provide the option if enhanced scopes are enabled
		if o.Flags.EnhancedScopes {
			if !o.IsFlagExplicitlySet(enhancedAPIFlagName) {
				if advanced {
					prompt := &survey.Confirm{
						Message: "Would you like to enable Cloud Build, Container Registry & Container Analysis APIs?",
						Default: o.Flags.EnhancedScopes,
						Help:    "Enables extra APIs on the GCP project",
					}
					err = survey.AskOne(prompt, &o.Flags.EnhancedApis, nil, surveyOpts)
					if err != nil {
						return err
					}
				} else {
					o.Flags.EnhancedApis = true
					log.Infof("Defaulting enabling Cloud Build, Container Registry & Container Analysis API's: %v", util.ColorPrompt(util.YesNo(o.Flags.EnhancedApis)))
				}
			}
		}
	}

	if o.Flags.EnhancedApis {
		log.Infof("checking if we need to enable APIs for GCB and GCR\n")

		err = gke.EnableAPIs(projectId, "cloudbuild", "containerregistry", "containeranalysis")
		if err != nil {
			return err
		}
	}

	if !o.BatchMode {
		// only provide the option if enhanced scopes are enabled
		if o.Flags.EnhancedScopes && !o.InstallOptions.Flags.Kaniko {
			if advanced {
				prompt := &survey.Confirm{
					Message: "Would you like to enable Kaniko for building container images",
					Default: o.Flags.EnhancedScopes,
					Help:    "Use Kaniko for docker images",
				}
				err = survey.AskOne(prompt, &o.InstallOptions.Flags.Kaniko, nil, surveyOpts)
				if err != nil {
					return err
				}
			} else {
				o.InstallOptions.Flags.Kaniko = false
				log.Infof("Defaulting enabling Kaniko for building container images: %v", util.ColorPrompt(util.YesNo(o.InstallOptions.Flags.Kaniko)))
			}
		}
	}

	if o.InstallOptions.Flags.NextGeneration || o.InstallOptions.Flags.Tekton || o.InstallOptions.Flags.Kaniko {
		// lets default the docker registry to GCR
		if o.InstallOptions.Flags.DockerRegistry == "" {
			o.InstallOptions.Flags.DockerRegistry = "gcr.io"
		}

		// lets default the docker registry org to the project id
		if o.InstallOptions.Flags.DockerRegistryOrg == "" {
			o.InstallOptions.Flags.DockerRegistryOrg = projectId
		}
	}

	// mandatory flags are machine type, num-nodes, zone or region
	args := []string{"container", "clusters", "create",
		clusterName,
		"--num-nodes", minNumOfNodes,
		"--machine-type", machineType,
		"--enable-autoscaling",
		"--min-nodes", minNumOfNodes,
		"--max-nodes", maxNumOfNodes}

	if region != "" {
		args = append(args, "--region", region)
	} else {
		args = append(args, "--zone", zone)
	}

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
		log.Infof("using cluster scopes: %s\n", util.ColorInfo(strings.Join(o.Flags.Scopes, " ")))

		args = append(args, fmt.Sprintf("--scopes=%s", strings.Join(o.Flags.Scopes, ",")))
	}

	if o.Flags.Preemptible {
		args = append(args, "--preemptible")
	}

	labels := o.Flags.Labels
	user, err := osUser.Current()
	if err == nil && user != nil {
		labels = addLabel(labels, "created-by", user.Username)
	}
	timeText := time.Now().Format("Mon-Jan-2-2006-15:04:05")
	labels = addLabel(labels, "create-time", timeText)
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
		o.InstallOptions.Flags.DefaultEnvironmentPrefix = clusterName
	}

	o.InstallOptions.setInstallValues(map[string]string{
		kube.Zone:        zone,
		kube.Region:      region,
		kube.ProjectID:   projectId,
		kube.ClusterName: clusterName,
	})

	err = o.initAndInstall(cloud.GKE)
	if err != nil {
		return err
	}

	getCredsCommand := []string{"container", "clusters", "get-credentials", clusterName}
	if "" != zone {
		getCredsCommand = append(getCredsCommand, "--zone", zone)
	} else if "" != region {
		getCredsCommand = append(getCredsCommand, "--region", region)
	}

	getCredsCommand = append(getCredsCommand, "--project", projectId)

	err = o.RunCommand("gcloud", getCredsCommand...)
	if err != nil {
		return err
	}

	context, err := o.GetCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}

	_, ns, err := o.KubeClientAndNamespace()
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

	return nil
}

// addLabel adds the given label key and value to the label string
func addLabel(labels string, name string, value string) string {
	username := sanitizeLabel(value)
	if username != "" {
		sep := ""
		if labels != "" {
			sep = ","
		}
		labels += sep + sanitizeLabel(name) + "=" + username
	}
	return labels
}

func sanitizeLabel(username string) string {
	sanitized := strings.ToLower(username)
	return disallowedLabelCharacters.ReplaceAllString(sanitized, "-")
}

// validateClusterName checks for compliance of a user supplied
// cluster name against GKE's rules for these names.
func validateClusterName(clustername string) error {
	// Check for length greater than 40.
	if len(clustername) > 40 {
		err := fmt.Errorf("cluster name %v is greater than the maximum 40 characters", clustername)
		return err
	}
	// Now we need only make sure that clustername is limited to
	// lowercase alphanumerics and dashes.
	if disallowedLabelCharacters.MatchString(clustername) {
		err := fmt.Errorf("cluster name %v contains invalid characters. Permitted are lowercase alphanumerics and `-`", clustername)
		return err
	}
	return nil
}
