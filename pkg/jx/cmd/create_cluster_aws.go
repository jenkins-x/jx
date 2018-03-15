package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
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

		jx create cluster aws

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

	cmd.Flags().BoolVarP(&options.Flags.UseRBAC, "rbac", "r", true, "whether to enable RBAC on the Kubernetes cluster")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "aws1", "The name of this cluster.")
	cmd.Flags().StringVarP(&options.Flags.NodeCount, optionNodes, "o", "", "node count")
	cmd.Flags().StringVarP(&options.Flags.KubeVersion, optionKubernetesVersion, "v", "", "kubernetes version")
	cmd.Flags().StringVarP(&options.Flags.Zones, optionZones, "z", "", "Availability zones. Defaults to $AWS_AVAILABILITY_ZONES")
	cmd.Flags().StringVarP(&options.Flags.InsecureDockerRegistry, "insecure-registry", "", "10.1.0.0/16", "The insecure docker registries to allow")
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

	nodeCount := flags.NodeCount
	if nodeCount == "" {
		prompt := &survey.Input{
			Message: "nodes",
			Default: "3",
			Help:    "number of nodes",
		}
		survey.AskOne(prompt, &nodeCount, nil)
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
			o.warnf("No AWS_AVAILABILITY_ZONES environment variable is defined or %s option!\n", optionZones)

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
	auth := "RBAC"
	if !flags.UseRBAC {
		auth = "AlwaysAllow"
	}
	args = append(args, "--authorization", auth, "--zones", zones, "--yes")

	// TODO allow add custom args?

	o.Printf("running command: %s\n", util.ColorInfo("kops "+strings.Join(args, " ")))
	err = o.runCommand("kops", args...)
	if err != nil {
		return err
	}

	o.Printf("\nKops has created cluster %s it will take a minute or so to startup\n", util.ColorInfo(name))
	o.Printf("You can check on the status in another terminal via the command: %s\n", util.ColorStatus("kops validate cluster"))
	o.Printf("Waiting for the kubernetes cluster to be ready so we can continue...\n")

	time.Sleep(30 * time.Second)

	insecureRegistries := flags.InsecureDockerRegistry
	if insecureRegistries != "" {
		igJson, err := o.waitForInstanceGroupJson()
		if err != nil {
			return fmt.Errorf("Failed to wait for the InstanceGroup YAML: %s\n", err)
		}
		o.Printf("Loaded nodes InstanceGroup JSON: %s\n", igJson)

		err = o.modifyInstanceGroupDockerConfig(igJson, insecureRegistries)
		if err != nil {
			return err
		}
	}

	err = o.waitForClusterToComeUp()
	if err != nil {
		return fmt.Errorf("Failed to wait for Kubernetes cluster to start: %s\n", err)
	}

	o.Printf("\n")
	o.runCommand("kops", "validate", "cluster")
	o.Printf("\n")

	return o.initAndInstall(AWS)
}

func (o *CreateClusterAWSOptions) waitForInstanceGroupJson() (string, error) {
	yamlOutput := ""
	f := func() error {
		text, err := o.getCommandOutput("", "kops", "get", "ig", "nodes", "-ojson")
		if err != nil {
			return err
		}
		yamlOutput = text
		return nil
	}
	err := o.retryQuiet(200, time.Second+10, f)
	if err == nil {
		lines := strings.Split(yamlOutput, "\n")
		for {
			if len(lines) == 0 {
				break
			}
			l1 := strings.TrimSpace(lines[0])
			if strings.HasPrefix(l1, "{") {
				break
			}
			lines = lines[1:]
		}
		yamlOutput = strings.Join(lines, "\n")
	}
	return yamlOutput, err
}

func (o *CreateClusterAWSOptions) waitForClusterToComeUp() error {
	f := func() error {
		_, err := o.getCommandOutput("", "kubectl", "get", "node")
		return err
	}
	return o.retryQuiet(200, time.Second+10, f)
}

func (o *CreateClusterAWSOptions) modifyInstanceGroupDockerConfig(json string, insecureRegistries string) error {
	if insecureRegistries == "" {
		return nil
	}
	newJson, err := kube.EnableInsecureRegistry(json, insecureRegistries)
	if err != nil {
		return fmt.Errorf("Failed to modify InstanceGroup JSON to add insecure registries %s: %s", insecureRegistries, err)
	}
	if newJson == json {
		return nil
	}
	o.Printf("new json: %s\n", newJson)
	tmpFile, err := ioutil.TempFile("", "kops-ig-json-")
	if err != nil {
		return err
	}
	fileName := tmpFile.Name()
	err = ioutil.WriteFile(fileName, []byte(newJson), DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write InstanceGroup JSON %s: %s", fileName, err)
	}

	o.Printf("Updating nodes InstanceGroup to enable insecure docker registries %s\n", util.ColorInfo(insecureRegistries))
	err = o.runCommand("kops", "replace", "-f", fileName)
	if err != nil {
		return err
	}

	o.Printf("Updating the cluster\n")
	err = o.runCommand("kops", "update", "cluster", "--yes")
	if err != nil {
		return err
	}

	o.Printf("Rolling update the cluster\n")
	err = o.runCommand("kops", "rolling-update", "cluster", "--cloudonly", "--yes")
	if err != nil {
		return err
	}
	return nil
}
