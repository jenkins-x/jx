package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/spf13/cobra"
)

type DeleteAwsOptions struct {
	CommonOptions

	VpcId  string
	Region string
}

func NewCmdDeleteAws(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteAwsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Deletes given AWS VPC and resources associated with it (like elastic load balancers or subnets)",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.VpcId, "vpc-id", "", "", "ID of VPC to delete.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use.")

	return cmd
}

func (o *DeleteAwsOptions) Run() error {
	vpcid := o.VpcId

	region := o.Region
	if region == "" {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
			if region == "" {
				region = "us-west-2"
			}
		}
	}
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(region)}))

	// Delete elastic load balancers assigned to VPC
	elbSvc := elbv2.New(session.New(&aws.Config{Region: aws.String(o.Region)}))
	loadBalancers, err := elbSvc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return err
	}
	for _, loadBalancer := range loadBalancers.LoadBalancers {
		if *loadBalancer.VpcId == vpcid {
			fmt.Printf("Deleting load balancer %s...\n", *loadBalancer.LoadBalancerName)
			_, err = elbSvc.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{LoadBalancerArn: loadBalancer.LoadBalancerArn})
			if err != nil {
				return err
			}
			fmt.Printf("Load balancer %s deleted.\n", *loadBalancer.LoadBalancerName)
		}
	}

	// Detached and delete internet gateways associated with given VPC
	internetGateways, err := svc.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("attachment.vpc-id"),
				Values: []*string{
					aws.String(vpcid),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, internetGateway := range internetGateways.InternetGateways {
		if len(internetGateway.Attachments) > 0 {
			_, err = svc.DetachInternetGateway(&ec2.DetachInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId, VpcId: aws.String(vpcid)})
			if err != nil {
				return err
			}
		}

		_, err = svc.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId})
		if err != nil {
			return err
		}
	}

	// Delete subnets assigned to VPC
	subnets, err := svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcid),
				},
			},
		},
	})
	for _, subnet := range subnets.Subnets {
		interfaces, err := svc.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("subnet-id"),
					Values: []*string{
						subnet.SubnetId,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		for _, iface := range interfaces.NetworkInterfaces {
			if iface.Attachment != nil {
				fmt.Printf("Detaching interface %s\n", *iface.NetworkInterfaceId)
				_, err = svc.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{AttachmentId: iface.Attachment.AttachmentId})
				if err != nil {
					return err
				}
			}

			_, err := svc.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: iface.NetworkInterfaceId})
			if err != nil {
				return err
			}
		}

		_, err = svc.DeleteSubnet(&ec2.DeleteSubnetInput{SubnetId: subnet.SubnetId})
		if err != nil {
			return err
		}
	}

	// Delete VPC
	_, err = svc.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(vpcid),
	})
	if err != nil {
		return err
	}

	return nil
}
