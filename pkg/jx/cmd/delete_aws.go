package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/log"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/spf13/cobra"
)

const gatewayDetachAttempts = 11

type DeleteAwsOptions struct {
	CommonOptions

	Profile string
	Region  string
	VpcId   string
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

	cmd.Flags().StringVarP(&options.LogLevel, "log-level", "", logger.InfoLevel.String(), "Logging level. Possible values - panic, fatal, error, warning, info, debug.")

	cmd.Flags().StringVarP(&options.Profile, "profile", "", "", "AWS profile to use.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use.")
	cmd.Flags().StringVarP(&options.VpcId, "vpc-id", "", "", "ID of VPC to delete.")

	return cmd
}

func (o *DeleteAwsOptions) Run() error {
	log.ConfigureLog(o.LogLevel)

	vpcid := o.VpcId

	session, err := amazon.NewAwsSession(o.Profile, o.Region)
	if err != nil {
		return err
	}
	svc := ec2.New(session)

	// Delete elastic load balancers assigned to VPC
	elbSvc := elbv2.New(session)
	loadBalancers, err := elbSvc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return err
	}
	for _, loadBalancer := range loadBalancers.LoadBalancers {
		if *loadBalancer.VpcId == vpcid {
			log.Infof("Deleting load balancer %s...\n", *loadBalancer.LoadBalancerName)
			_, err = elbSvc.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{LoadBalancerArn: loadBalancer.LoadBalancerArn})
			if err != nil {
				return err
			}
			log.Infof("Load balancer %s deleted.\n", *loadBalancer.LoadBalancerName)
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
			detachAttemptsLeft := gatewayDetachAttempts
			for ; detachAttemptsLeft > 0; {
				_, err = svc.DetachInternetGateway(&ec2.DetachInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId, VpcId: aws.String(vpcid)})
				log.Infof("Detaching internet gateway %s from VPC %s...\n", *internetGateway.InternetGatewayId, vpcid)
				if err != nil {
					if strings.Contains(err.Error(), "Please unmap those public address(es) before detaching the gateway") {
						detachAttemptsLeft--
						log.Infof("Waiting for public address to be unmapped from internet gateway. Detach attempts left: %d\n", detachAttemptsLeft)
						time.Sleep(10 * time.Second)
					} else {
						return err
					}
				} else {
					detachAttemptsLeft = 0
				}
			}
			log.Infof("Internet gateway %s detached successfully from VPC %s...\n", *internetGateway.InternetGatewayId, vpcid)
		}

		_, err = svc.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId})
		log.Infof("Deleting internet gateway %s...\n", *internetGateway.InternetGatewayId)
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
				log.Infof("Detaching interface %s\n", *iface.NetworkInterfaceId)
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
