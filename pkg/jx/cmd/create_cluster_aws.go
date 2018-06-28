package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	optionZones = "zones"
)

// CreateClusterAWSOptions contains the CLI flags
type CreateClusterAWSOptions struct {
	CreateClusterOptions

	Flags CreateClusterAWSFlags
}

type CreateClusterAWSFlags struct {
	ClusterName            string
	NodeCount              string
	KubeVersion            string
	Zones                  string
	InsecureDockerRegistry string
	UseRBAC                bool
	TerraformDirectory     string
	NodeSize               string
	MasterSize             string
}

var (
	createClusterAWSLong = templates.LongDesc(`
		This command creates a new kubernetes cluster on Amazon Web Service (AWS) using kops, installing required local dependencies and provisions the
		Jenkins X platform

		AWS manages your hosted Kubernetes environment via kops, making it quick and easy to deploy and
		manage containerized applications without container orchestration expertise. It also eliminates the burden of
		ongoing operations and maintenance by provisioning, upgrading, and scaling resources on demand, without taking
		your applications offline.

`)

	createClusterAWSExample = templates.Examples(`
        # to create a new kubernetes cluster with Jenkins X in your default zones (from $AWS_AVAILABILITY_ZONES)
		jx create cluster aws

		# to specify the zones
		jx create cluster aws --zones us-west-2a,us-west-2b,us-west-2c

		# to output terraform configuration
		jx create cluster aws --terraform /Users/jx/jx-infra
`)
)

// NewCmdCreateClusterAWS creates the command
func NewCmdCreateClusterAWS(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterAWSOptions{
		CreateClusterOptions: createCreateClusterOptions(f, out, errOut, AKS),
	}
	cmd := &cobra.Command{
		Use:     "aws",
		Short:   "Create a new kubernetes cluster on AWS with kops",
		Long:    createClusterAWSLong,
		Example: createClusterAWSExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	cmd.Flags().BoolVarP(&options.Flags.UseRBAC, "rbac", "r", true, "whether to enable RBAC on the Kubernetes cluster")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "aws1", "The name of this cluster.")
	cmd.Flags().StringVarP(&options.Flags.NodeCount, optionNodes, "o", "", "node count")
	cmd.Flags().StringVarP(&options.Flags.KubeVersion, optionKubernetesVersion, "v", "", "kubernetes version")
	cmd.Flags().StringVarP(&options.Flags.Zones, optionZones, "z", "", "Availability zones. Defaults to $AWS_AVAILABILITY_ZONES")
	cmd.Flags().StringVarP(&options.Flags.InsecureDockerRegistry, "insecure-registry", "", "100.64.0.0/10", "The insecure docker registries to allow")
	cmd.Flags().StringVarP(&options.Flags.TerraformDirectory, "terraform", "t", "", "The directory to save terraform configuration.")
	cmd.Flags().StringVarP(&options.Flags.NodeSize, "node-size", "", "", "The size of a node in the kops created cluster.")
	cmd.Flags().StringVarP(&options.Flags.MasterSize, "master-size", "", "", "The size of a master in the kops created cluster.")
	return cmd
}

// Run runs the command
func (o *CreateClusterAWSOptions) Run() error {
	var deps []string
	d := binaryShouldBeInstalled("kops")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("%v\nPlease fix the error or install manually then try again", err)
		os.Exit(-1)
	}

	flags := &o.Flags

	if flags.NodeCount == "" {
		prompt := &survey.Input{
			Message: "nodes",
			Default: "3",
			Help:    "number of nodes",
		}
		survey.AskOne(prompt, &flags.NodeCount, nil)
	}

	/*
		kubeVersion := o.Flags.KubeVersion
		if kubeVersion == "" {
			prompt := &survey.Input{
				Message: "Kubernetes version",
				Default: kubeVersion,
				Help:    "The release version of kubernetes to install in the cluster",
			}
			survey.AskOne(prompt, &kubeVersion, nil)
		}
	*/

	zones := flags.Zones
	if zones == "" {
		zones = os.Getenv("AWS_AVAILABILITY_ZONES")
		if zones == "" {
			log.Warnf("No AWS_AVAILABILITY_ZONES environment variable is defined or %s option!\n", optionZones)

			prompt := &survey.Input{
				Message: "Availability zones",
				Default: "",
				Help:    "The AWS Availability Zones to use for the Kubernetes cluster",
			}
			err = survey.AskOne(prompt, &zones, survey.Required)
			if err != nil {
				return err
			}
		}
	}
	if zones == "" {
		return fmt.Errorf("No Availility zones provided!")
	}

	name := flags.ClusterName
	if name == "" {
		name = "aws1"
	}
	if !strings.Contains(name, ".") {
		name = name + ".cluster.k8s.local"
	}

	args := []string{"create", "cluster", "--name", name}
	if flags.NodeCount != "" {
		args = append(args, "--node-count", flags.NodeCount)
	}
	if flags.KubeVersion != "" {
		args = append(args, "--kubernetes-version", flags.KubeVersion)
	}

	if flags.NodeSize != "" {
		args = append(args, "--node-size", flags.NodeSize)
	}
	if flags.MasterSize != "" {
		args = append(args, "--master-size", flags.MasterSize)
	}

	auth := "RBAC"
	if !flags.UseRBAC {
		auth = "AlwaysAllow"
	}
	args = append(args, "--authorization", auth, "--zones", zones, "--yes")

	if flags.TerraformDirectory != "" {
		args = append(args, "--out", flags.TerraformDirectory, "--target=terraform")
	}

	// TODO allow add custom args?
	log.Info("Creating cluster...\n")
	log.Infof("running command: %s\n", util.ColorInfo("kops "+strings.Join(args, " ")))
	err = o.runCommandVerbose("kops", args...)
	if err != nil {
		return err
	}

	log.Infof("\nKops has created cluster %s it will take a minute or so to startup\n", util.ColorInfo(name))
	log.Infof("You can check on the status in another terminal via the command: %s\n", util.ColorStatus("kops validate cluster"))

	time.Sleep(5 * time.Second)

	insecureRegistries := flags.InsecureDockerRegistry
	if insecureRegistries != "" {
		log.Warn("Waiting for the Cluster configuration...")
		igJson, err := o.waitForClusterJson(name)
		if err != nil {
			return fmt.Errorf("Failed to wait for the Cluster JSON: %s\n", err)
		}
		log.Infof("Loaded Cluster JSON: %s\n", igJson)

		err = o.modifyClusterConfigJson(igJson, insecureRegistries)
		if err != nil {
			return err
		}
		log.Infoln("Cluster configuration updated")
	}

	log.Infoln("Waiting for the kubernetes cluster to be ready so we can continue...")
	err = o.waitForClusterToComeUp()
	if err != nil {
		return fmt.Errorf("Failed to wait for Kubernetes cluster to start: %s\n", err)
	}

	log.Blank()
	log.Infoln("Validating kops cluster state...")
	err = o.runCommand("kops", "validate", "cluster")
	if err != nil {
		return fmt.Errorf("Failed to successfully validate kops cluster state: %s\n", err)
	}
	log.Infoln("State of kops cluster: OK")
	log.Blank()

	log.Info("Initialising cluster ...\n")
	return o.initAndInstall(AWS)
}

func (o *CreateClusterAWSOptions) waitForClusterJson(clusterName string) (string, error) {
	jsonOutput := ""
	f := func() error {
		text, err := o.getCommandOutput("", "kops", "get", "cluster", clusterName, "-ojson")
		if err != nil {
			return err
		}
		jsonOutput = text
		return nil
	}
	err := o.retryQuiet(200, time.Second*10, f)
	return jsonOutput, err
}

func (o *CreateClusterAWSOptions) waitForClusterToComeUp() error {
	f := func() error {
		return o.runCommandQuietly("kubectl", "get", "node")
	}
	return o.retryQuiet(2000, time.Second*10, f)
}

func (o *CreateClusterAWSOptions) modifyClusterConfigJson(json string, insecureRegistries string) error {
	if insecureRegistries == "" {
		return nil
	}
	newJson, err := kube.EnableInsecureRegistry(json, insecureRegistries)
	if err != nil {
		return fmt.Errorf("Failed to modify Cluster JSON to add insecure registries %s: %s", insecureRegistries, err)
	}
	if newJson == json {
		return nil
	}
	log.Infof("new json: %s\n", newJson)
	tmpFile, err := ioutil.TempFile("", "kops-ig-json-")
	if err != nil {
		return err
	}
	fileName := tmpFile.Name()
	err = ioutil.WriteFile(fileName, []byte(newJson), DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write InstanceGroup JSON %s: %s", fileName, err)
	}

	log.Infof("Updating Cluster configuration to enable insecure docker registries %s\n", util.ColorInfo(insecureRegistries))
	err = o.runCommand("kops", "replace", "-f", fileName)
	if err != nil {
		return err
	}

	log.Infoln("Updating the cluster")
	err = o.runCommand("kops", "update", "cluster", "--yes")
	if err != nil {
		return err
	}

	log.Infoln("Rolling update the cluster")
	err = o.runCommand("kops", "rolling-update", "cluster", "--cloudonly", "--yes")
	if err != nil {
		// lets not fail to install if the rolling upgrade fails
		log.Warnf("Failed to perform rolling upgrade: %s\n", err)
		//return err
	}
	return nil
}
