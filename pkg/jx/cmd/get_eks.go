package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

type GetEksOptions struct {
	GetOptions
	Profile string
	Region  string
}

var (
	getEksLong = templates.LongDesc(`
		Display one or many EKS cluster resources 
`)

	getEksExample = templates.Examples(`
		# List EKS clusters available in AWS
		jx get eks

		# Displays someCluster EKS resource
		jx get eks someCluster

		# Displays someCluster resource in YAML format
		jx get eks someCluster -oyaml
	`)
)

func NewCmdGetEks(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetEksOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "eks",
		Short:   "List EKS clusters.",
		Long:    getEksLong,
		Example: getEksExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Profile, "profile", "", "", "AWS profile to use.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use. Default: "+amazon.DefaultRegion)

	options.addGetFlags(cmd)
	return cmd
}

func (o *GetEksOptions) Run() error {
	if len(o.Args) == 0 {
		var deps []string
		d := opts.BinaryShouldBeInstalled("eksctl")
		if d != "" {
			deps = append(deps, d)
		}
		d = opts.BinaryShouldBeInstalled("heptio-authenticator-aws")
		if d != "" {
			deps = append(deps, d)
		}
		err := o.InstallMissingDependencies(deps)
		if err != nil {
			log.Errorf("%v\nPlease fix the error or install manually then try again", err)
			os.Exit(-1)
		}

		region, err := amazon.ResolveRegion(o.Profile, o.Region)
		if err != nil {
			return err
		}
		cmd := exec.Command("eksctl", "get", "cluster", "--region", region)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil
		}
		fmt.Print(string(output))
		return nil
	} else {
		cluster := o.Args[0]
		session, err := amazon.NewAwsSession(o.Profile, o.Region)
		if err != nil {
			return err
		}
		svc := ec2.New(session)
		instances, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("tag:eksctl.cluster.k8s.io/v1alpha1/cluster-name"),
					Values: []*string{
						aws.String(cluster),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		if o.Output == "" {
			fmt.Println("NAME")
			fmt.Println(cluster)
		} else if o.Output == "yaml" {
			reservations, err := yaml.Marshal(instances.Reservations)
			if err != nil {
				return err
			}
			fmt.Println(string(reservations))
		} else {
			return errors.New("Invalid output format.")
		}

		return nil
	}
}
