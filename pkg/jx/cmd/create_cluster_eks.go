package cmd

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// CreateClusterEKSOptions contains the CLI flags
type CreateClusterEKSOptions struct {
	CreateClusterOptions

	Flags CreateClusterEKSFlags
}

type CreateClusterEKSFlags struct {
	ClusterName  string
	NodeCount    int
	NodesMin     int
	NodesMax     int
	Region       string
	Profile      string
	SshPublicKey string
}

var (
	createClusterEKSLong = templates.LongDesc(`
		This command creates a new kubernetes cluster on Amazon Web Services (AWS) using EKS, installing required local dependencies and provisions the
		Jenkins X platform

		EKS is a managed kubernetes service on AWS.

`)

	createClusterEKSExample = templates.Examples(`
        # to create a new kubernetes cluster with Jenkins X in your default zones (from $EKS_AVAILABILITY_ZONES)
		jx create cluster eks

		# to specify the zones
		jx create cluster eks --zones us-west-2a,us-west-2b,us-west-2c
`)
)

// NewCmdCreateClusterEKS creates the command
func NewCmdCreateClusterEKS(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := CreateClusterEKSOptions{
		CreateClusterOptions: createCreateClusterOptions(f, out, errOut, AKS),
	}
	cmd := &cobra.Command{
		Use:     "eks",
		Short:   "Create a new kubernetes cluster on AWS using EKS",
		Long:    createClusterEKSLong,
		Example: createClusterEKSExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCreateClusterFlags(cmd)
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "eks1", "The name of this cluster.")
	cmd.Flags().IntVarP(&options.Flags.NodeCount, optionNodes, "o", 0, "number of nodes")
	cmd.Flags().IntVarP(&options.Flags.NodesMin, "nodes-min", "", -1, "minimum number of nodes")
	cmd.Flags().IntVarP(&options.Flags.NodesMax, "nodes-max", "", -1, "maximum number of nodes")
	cmd.Flags().StringVarP(&options.Flags.Region, "region", "r", "us-west-2", "The region to use.")
	cmd.Flags().StringVarP(&options.Flags.Profile, "profile", "p", "", "AWS profile to use. If provided, this overrides the AWS_PROFILE environment variable")
	cmd.Flags().StringVarP(&options.Flags.SshPublicKey, "ssh-public-key", "", "", "SSH public key to use for nodes (import from local path, or use existing EC2 key pair) (default \"~/.ssh/id_rsa.pub\")")
	return cmd
}

// Run runs the command
func (o *CreateClusterEKSOptions) Run() error {
	var deps []string
	/*
		d := binaryShouldBeInstalled("aws")
			if d != "" {
				deps = append(deps, d)
			}
	*/
	d := binaryShouldBeInstalled("eksctl")
	if d != "" {
		deps = append(deps, d)
	}
	d = binaryShouldBeInstalled("heptio-authenticator-aws")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("%v\nPlease fix the error or install manually then try again", err)
		os.Exit(-1)
	}

	flags := &o.Flags

	args := []string{"create", "cluster"}
	if flags.ClusterName != "" {
		args = append(args, "--cluster-name", flags.ClusterName)
	}
	if flags.Region != "" {
		args = append(args, "--region", flags.Region)
	}
	if flags.Profile != "" {
		args = append(args, "--profile", flags.Profile)
	}
	if flags.SshPublicKey != "" {
		args = append(args, "--ssh-public-key", flags.SshPublicKey)
	}
	if flags.NodeCount >= 0 {
		args = append(args, "--node-count", strconv.Itoa(flags.NodeCount))
	}
	if flags.NodesMin >= 0 {
		args = append(args, "--nodes-min", strconv.Itoa(flags.NodesMin))
	}
	if flags.NodesMax >= 0 {
		args = append(args, "--nodes-max", strconv.Itoa(flags.NodesMax))
	}

	log.Info("Creating EKS cluster - this can take a while so please be patient...\n")
	log.Infof("running command: %s\n", util.ColorInfo("eksctl "+strings.Join(args, " ")))
	err = o.runCommandVerbose("eksctl", args...)
	if err != nil {
		return err
	}
	log.Blank()

	log.Info("Initialising cluster ...\n")
	return o.initAndInstall(EKS)
}
