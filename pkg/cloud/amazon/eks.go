package amazon

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"os/exec"
	"strings"
)

// EksClusterExists checks if EKS cluster with given name exists in given region.
func EksClusterExists(clusterName string, profile string, region string) (bool, error) {
	region, err := ResolveRegion(profile, region)
	if err != nil {
		return false, err
	}
	cmd := exec.Command("eksctl", "get", "cluster", "--region", region)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	for i, line := range strings.Split(string(output), "\n") {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(line, clusterName + "\t") {
			return true, nil
		}
	}

	return false, nil
}


// EksClusterObsoleteStackExists detects if there is obsolete CloudFormation stack for given EKS cluster.
//
// If EKS cluster creation process is interrupted, there will be CloudFormation stack in ROLLBACK_COMPLETE state left.
// Such dead stack prevents eksctl from creating cluster with the same name. This is common activity then to remove stacks
// like this and this function performs this action.
func EksClusterObsoleteStackExists(clusterName string, profile string, region string) (bool, error)  {
	session, err := NewAwsSession(profile, region)
	if err != nil {
		return false, err
	}
	cloudformationService := cloudformation.New(session)
	stacks, err := cloudformationService.ListStacks(&cloudformation.ListStacksInput{
		StackStatusFilter: []*string{aws.String("ROLLBACK_COMPLETE")},
	})
	if err != nil {
		return false, err
	}
	for _, stack := range stacks.StackSummaries {
		if *stack.StackName == EksctlStackName(clusterName) {
			return true, nil
		}
	}

	return false, nil
}

// CleanUpObsoleteEksClusterStack removes dead eksctl CloudFormation stack associated with given EKS cluster name.
func CleanUpObsoleteEksClusterStack(clusterName string, profile string, region string) error {
	session, err := NewAwsSession(profile, region)
	if err != nil {
		return err
	}
	cloudformationService := cloudformation.New(session)
	_, err = cloudformationService.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(EksctlStackName(clusterName)),
	})

	return err
}

// EksctlStackName generates CloudFormation stack name for given EKS cluster name. This function follows eksctl
// naming convention.
func EksctlStackName(clusterName string) string {
	return fmt.Sprintf("eksctl-%s-cluster", clusterName)
}