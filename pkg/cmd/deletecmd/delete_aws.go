package deletecmd

import (
	"strings"
	"time"

	session2 "github.com/jenkins-x/jx/pkg/cloud/amazon/session"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/spf13/cobra"
)

const gatewayDetachAttempts = 10

type DeleteAwsOptions struct {
	*opts.CommonOptions

	Profile string
	Region  string
	VpcId   string
}

func NewCmdDeleteAws(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &DeleteAwsOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Deletes given AWS VPC and resources associated with it (like elastic load balancers or subnets)",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Profile, "profile", "", "", "AWS profile to use.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use.")
	cmd.Flags().StringVarP(&options.VpcId, "vpc-id", "", "", "ID of VPC to delete.")

	return cmd
}

func (o *DeleteAwsOptions) Run() error {
	vpcid := o.VpcId

	session, err := session2.NewAwsSession(o.Profile, o.Region)
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
			log.Logger().Infof("Deleting load balancer %s...", *loadBalancer.LoadBalancerName)
			_, err = elbSvc.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{LoadBalancerArn: loadBalancer.LoadBalancerArn})
			if err != nil {
				return err
			}
			log.Logger().Infof("Load balancer %s deleted.", *loadBalancer.LoadBalancerName)
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
			err = o.RetryUntilFatalError(gatewayDetachAttempts, 10*time.Second, func() (fatalError *opts.FatalError, e error) {
				_, err = svc.DetachInternetGateway(&ec2.DetachInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId, VpcId: aws.String(vpcid)})
				log.Logger().Infof("Detaching internet gateway %s from VPC %s...", *internetGateway.InternetGatewayId, vpcid)
				if err != nil {
					if strings.Contains(err.Error(), "Please unmap those public address(es) before detaching the gateway") {
						log.Logger().Info("Waiting for public address to be unmapped from internet gateway.")
						return nil, err
					}
					return &opts.FatalError{E: err}, nil
				}
				return nil, nil
			})
			if err != nil {
				return err
			}
			log.Logger().Infof("Internet gateway %s detached successfully from VPC %s...", *internetGateway.InternetGatewayId, vpcid)
		}

		_, err = svc.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{InternetGatewayId: internetGateway.InternetGatewayId})
		log.Logger().Infof("Deleting internet gateway %s...", *internetGateway.InternetGatewayId)
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
				log.Logger().Infof("Detaching interface %s", *iface.NetworkInterfaceId)
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
